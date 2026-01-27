package brokcli

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/vela-ssoc/ssoc-proto/muxconn"
	"github.com/vela-ssoc/ssoc-proto/muxproto"
)

type Options struct {
	Addresses  []string
	Secret     string
	Semver     string
	Handler    http.Handler
	Validator  func(any) error
	DialConfig muxconn.DialConfig
}

func Open(ctx context.Context, opts Options) (Muxer, error) {
	hostname, _ := os.Hostname()
	req := &authRequest{
		Secret:   opts.Secret,
		Semver:   opts.Semver,
		Goos:     runtime.GOOS,
		Goarch:   runtime.GOARCH,
		Hostname: hostname,
	}

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
	req  *authRequest
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
	dc := bc.opts.DialConfig
	mux, err := dc.DialContext(bc.ctx, bc.opts.Addresses)
	if err != nil {
		return err
	}

	// 握手认证
	cfg, err := bc.authentication(mux)
	if err != nil {
		return err
	}
	bc.mux.store(mux, *cfg)

	return nil
}

//goland:noinspection GoUnhandledErrorResult
func (bc *brokClient) authentication(mux muxconn.Muxer) (*BrokConfig, error) {
	outbound := muxproto.Outbound(mux.Addr())
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
	if err = muxproto.WriteAuth(conn, bc.req); err != nil {
		attrs = append(attrs, "error", err)
		bc.log().Warn("写入认证报文出错", attrs...)
		return nil, err
	}

	resp := new(authResponse)
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	if err = muxproto.ReadAuth(conn, resp); err != nil {
		attrs = append(attrs, "error", err)
		bc.log().Warn("读取认证报文出错", attrs...)
		return nil, err
	}
	if !resp.isSucceed() {
		bc.log().Warn("服务端认证失败", attrs...)
		return nil, resp
	}

	cfg := resp.Config
	if err = bc.validBrokConfig(cfg); err != nil {
		bc.log().Warn("服务端响应的配置校验出错", attrs...)
		return nil, err
	}
	bc.log().Info("通道认证成功", attrs...)

	return cfg, nil
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
		bc.log().Error("客户端通道断线了", "error", err)

		// 开始重连
		if err = bc.openLoop(); err != nil {
			break
		}

		bc.log().Info("客户端通道重连成功")
	}
}

func (bc *brokClient) log() *slog.Logger {
	if l := bc.opts.DialConfig.Logger; l != nil {
		return l
	}

	return slog.Default()
}

func (bc *brokClient) perTimeout() time.Duration {
	if d := bc.opts.DialConfig.PerTimeout; d > 0 {
		return d
	}

	return 10 * time.Second
}

func (bc *brokClient) perContext() (context.Context, context.CancelFunc) {
	d := bc.perTimeout()

	return context.WithTimeout(bc.ctx, d)
}

func (bc *brokClient) validBrokConfig(cfg *BrokConfig) error {
	if v := bc.opts.Validator; v != nil {
		return v(cfg)
	}

	if cfg.DSN == "" {
		return errors.New("响应报文缺少数据库连接地址(dsn)")
	}

	return nil
}

func (*brokClient) backoffDuration(startAt time.Time, retries int) time.Duration {
	if retries <= 10 {
		return 2 * time.Second
	} else if retries <= 100 {
		return 5 * time.Second
	} else if retries <= 200 {
		return 10 * time.Minute
	} else if retries <= 500 {
		return 10 * time.Minute
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
