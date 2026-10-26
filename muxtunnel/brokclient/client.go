package brokclient

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	"github.com/vela-ssoc/vela-tunnel/muxconn"
)

type Options struct {
	Addresses  []string
	Secret     string
	Version    string
	Handler    http.Handler
	Validator  func(any) error
	DialConfig *muxconn.DialWebsocketOptions
}

func Open(ctx context.Context, opts Options) (Muxer, error) {
	hostname, _ := os.Hostname()
	req := &muxproto.BrokerAuthRequest{
		Secret:   opts.Secret,
		Version:  opts.Version,
		Goos:     runtime.GOOS,
		Goarch:   runtime.GOARCH,
		PID:      os.Getpid(),
		Hostname: hostname,
		Args:     os.Args,
	}
	req.Workdir, _ = os.Getwd()
	req.Executable, _ = os.Executable()

	mux := new(safeMUX)
	cli := &brokClient{opts: opts, req: req, mux: mux, ctx: ctx}
	if err := cli.openLoop(); err != nil {
		return nil, err
	}

	// 连接成功
	go cli.serveHTTP()

	return mux, nil
}

type brokClient struct {
	opts Options
	req  *muxproto.BrokerAuthRequest
	mux  *safeMUX
	ctx  context.Context
}

// open 方法会一直尝试连接支持成功或遇到不可重试的错误。
func (bc *brokClient) openLoop() error {
	var retries int
	startAt := time.Now()
	for {
		retries++
		attrs := []any{"start_at", startAt, "retries", retries}
		err := bc.open()
		if err == nil {
			return nil
		}

		du := bc.backoffDuration(startAt, retries)
		attrs = append(attrs, "error", err, "sleep", du.String())
		bc.log().Warn("通道连接失败，稍后重试", attrs...)

		if cause := bc.sleep(du); cause != nil {
			attrs = append(attrs, "cause", cause)
			bc.log().Error("不可重试的错误", attrs...)
			return cause
		}
	}
}

func (bc *brokClient) open() error {
	addrs := bc.opts.Addresses
	mux, err := muxconn.Dial(bc.ctx, addrs, bc.opts.DialConfig)
	if err != nil {
		return err
	}

	// 握手认证
	cfg, err := bc.authentication(mux)
	if err != nil {
		_ = mux.Close()
		return err
	}
	bc.mux.store(mux, cfg)

	return nil
}

//goland:noinspection GoUnhandledErrorResult
func (bc *brokClient) authentication(mux muxconn.Muxer) (*model.BrokerConfig, error) {
	outbound := bc.outbound(mux.Addr())
	attrs := []any{"outbound", outbound}
	bc.req.Inet = outbound.String()

	ctx, cancel := bc.perContext()
	conn, err := mux.Open(ctx)
	cancel()
	if err != nil {
		attrs = append(attrs, "error", err)
		bc.log().Error("打开认证虚拟通道出错", attrs...)
		return nil, err
	}
	defer conn.Close()

	timeout := bc.perTimeout()
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	if err = muxconn.WriteAuth(conn, bc.req); err != nil {
		attrs = append(attrs, "error", err)
		bc.log().Warn("写入认证报文出错", attrs...)
		return nil, err
	}

	bcfg := new(model.BrokerConfig)
	resp := &muxproto.BrokerAuthResponse{Config: bcfg}
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	if err = muxconn.ReadAuth(conn, resp); err != nil {
		attrs = append(attrs, "error", err)
		bc.log().Warn("读取认证报文出错", attrs...)
		return nil, err
	}
	if re := resp.Err(); re != nil {
		attrs = append(attrs, "error", re)
		bc.log().Warn("服务端认证失败", attrs...)
		return nil, re
	}

	if err = bc.validBrokConfig(bcfg); err != nil {
		bc.log().Warn("服务端响应的配置校验出错", attrs...)
		return nil, err
	}
	bc.log().Info("通道认证成功", attrs...)

	return bcfg, nil
}

func (bc *brokClient) serveHTTP() {
	h := bc.opts.Handler
	if h == nil {
		h = http.NotFoundHandler()
	}

	for {
		srv := &http.Server{Handler: h}
		err := srv.Serve(bc.mux)
		_ = bc.mux.Close() // 确保关闭

		if cause := bc.ctx.Err(); cause != nil {
			bc.log().Error("客户端通道退出连接", "error", err, "cause", cause)
			break
		}
		// 开始重连
		bc.log().Warn("客户端通道掉线了", "error", err)
		if err = bc.openLoop(); err != nil {
			break
		}

		bc.log().Info("客户端通道重连成功")
	}
}

func (bc *brokClient) log() *slog.Logger {
	if dc := bc.opts.DialConfig; dc != nil && dc.Logger != nil {
		return dc.Logger
	}

	return slog.Default()
}

func (bc *brokClient) perTimeout() time.Duration {
	if dc := bc.opts.DialConfig; dc != nil && dc.PerTimeout > 0 {
		return dc.PerTimeout
	}

	return 10 * time.Second
}

func (bc *brokClient) perContext() (context.Context, context.CancelFunc) {
	d := bc.perTimeout()

	return context.WithTimeout(bc.ctx, d)
}

func (bc *brokClient) validBrokConfig(cfg *model.BrokerConfig) error {
	if v := bc.opts.Validator; v != nil {
		return v(cfg)
	}

	cd := cfg.Database
	if cd == nil {
		return errors.New("响应报文缺少数据库配置")
	}
	if cd.URI == "" {
		return errors.New("响应报文缺少 MongoDB 数据库配置")
	}
	if cd.DSN == "" {
		return errors.New("响应报文缺少关系数据库配置")
	}

	return nil
}

func (*brokClient) backoffDuration(startAt time.Time, retries int) time.Duration {
	if retries <= 10 {
		return 2 * time.Second
	} else if retries <= 100 {
		return 5 * time.Second
	} else if retries <= 200 {
		return 10 * time.Second
	} else if retries <= 500 {
		return 30 * time.Second
	}

	return time.Minute
}

func (bc *brokClient) sleep(d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-bc.ctx.Done():
		return bc.ctx.Err()
	}
}

func (bc *brokClient) outbound(laddr net.Addr) net.IP {
	if ta, ok := laddr.(*net.TCPAddr); ok {
		return ta.IP
	}

	return net.IPv4zero
}
