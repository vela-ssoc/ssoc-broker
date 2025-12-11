package clientd

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/vela-ssoc/ssoc-common/linkhub"
	"github.com/xtaci/smux"
)

func Open(ctx context.Context, cfg Config, opt Options) (Muxer, error) {
	if err := cfg.preparse(); err != nil {
		return nil, err
	}

	mux := new(safeMuxer)
	bc := &brokerClient{
		cfg: cfg,
		opt: opt,
		mux: mux,
		ctx: ctx,
	}
	if err := bc.open(); err != nil {
		return nil, err
	}
	go bc.serve()

	return mux, nil
}

type brokerClient struct {
	cfg Config
	opt Options
	mux *safeMuxer
	ctx context.Context
}

// open 持续尝试连接 manager 直至成功或遇到不可重试错误。
func (bc *brokerClient) open() error {
	addrs := bc.cfg.Addresses
	timeout := bc.opt.timeout()
	attrs := []any{slog.Any("addresses", addrs), slog.Duration("timeout", timeout)}
	bc.opt.logger().Info("开始连接认证", attrs...)

	beginAt := time.Now()
	var fails int
	for {
		sess, resp, err := bc.connects(addrs, timeout)
		if err == nil {
			bc.mux.replace(sess, resp.BootConfig)
			bc.opt.logger().Info("通道连接认证成功", attrs...)
			return nil
		}

		fails++
		du := bc.waitN(fails, beginAt)
		msgs := append(attrs, slog.Time("begin_at", beginAt))
		msgs = append(msgs, slog.Int("fails", fails))
		msgs = append(msgs, slog.Duration("sleep", du))
		msgs = append(msgs, slog.Any("error", err))
		bc.opt.logger().Warn("通道连接或认证失败，稍后重试", attrs...)
		if err = bc.sleep(du); err != nil {
			attrs = append(attrs, slog.Any("sleep_error", err))
			bc.opt.logger().Error("context 取消，退出重连机制", attrs...)
			return err
		}
	}
}

func (bc *brokerClient) connects(addrs []string, timeout time.Duration) (*smux.Session, *authResponse, error) {
	var errs []error
	for _, addr := range addrs {
		sess, resp, err := bc.connect(addr, timeout)
		if err == nil {
			return sess, resp, nil
		}
		errs = append(errs, err)
	}

	return nil, nil, errors.Join(errs...)
}

func (bc *brokerClient) connect(addr string, timeout time.Duration) (*smux.Session, *authResponse, error) {
	sess, err := bc.openSMUX(addr, timeout)
	if err != nil {
		bc.opt.logger().Warn("基础 TCP 网络连接错误", "addr", addr, "error", err)
		return nil, nil, err
	}

	bc.opt.logger().Info("基础 TCP 网络连接成功", "addr", addr)

	resp, err := bc.authentication(sess, timeout)
	if err != nil {
		_ = sess.Close()
		return nil, nil, err
	}
	if err = resp.checkError(); err != nil {
		_ = sess.Close()
		bc.opt.logger().Warn("上线认证失败", "error", err)
		return nil, nil, err
	}
	bc.opt.logger().Info("上线认证成功")

	return sess, resp, nil
}

func (bc *brokerClient) openSMUX(addr string, timeout time.Duration) (*smux.Session, error) {
	dialer := bc.opt.dialer()

	ctx, cancel := context.WithTimeout(bc.ctx, timeout)
	defer cancel()

	destURL := &url.URL{Scheme: "ws", Host: addr, Path: "/api/v1/tunnel"}
	ws, _, err := dialer.DialContext(ctx, destURL.String(), nil)
	if err != nil {
		return nil, err
	}
	conn := ws.NetConn()
	sess, err := smux.Client(conn, nil)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return sess, nil
}

func (bc *brokerClient) authentication(sess *smux.Session, timeout time.Duration) (*authResponse, error) {
	pre, err := sess.OpenStream()
	if err != nil {
		bc.opt.logger().Warn("打开认证虚拟子流错误", "error", err)
		return nil, err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer pre.Close()
	_ = pre.SetDeadline(time.Now().Add(timeout))

	req := &authRequest{Secret: bc.cfg.Secret, Semver: bc.cfg.Semver}
	if err = linkhub.WriteAuth(pre, req); err != nil {
		bc.opt.logger().Warn("写入认证请求数据出错", "error", err)
		return nil, err
	}
	bc.opt.logger().Info("写入认证请求数据成功")

	resp := new(authResponse)
	if err = linkhub.ReadAuth(pre, resp); err != nil {
		bc.opt.logger().Warn("读取认证响应数据出错", "error", err)
		return nil, err
	}
	bc.opt.logger().Info("读取认证响应数据成功")

	return resp, nil
}

func (bc *brokerClient) serve() {
	for {
		hand := bc.opt.handler()
		srv := &http.Server{
			Handler:      hand,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  30 * time.Second,
		}
		lis := &muxerListener{mux: bc.mux}
		err := srv.Serve(lis)
		_ = lis.Close()

		const sleep = 3 * time.Second
		bc.opt.logger().Warn("broker 掉线了", "error", err, "sleep", sleep)
		_ = bc.sleep(sleep)
		if err = bc.open(); err != nil {
			break
		}
	}
}

func (bc *brokerClient) sleep(d time.Duration) error {
	ctx := bc.ctx
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func (*brokerClient) waitN(fails int, startAt time.Time) time.Duration {
	if fails <= 30 {
		return 2 * time.Second
	} else if fails <= 100 {
		return 5 * time.Second
	} else if fails <= 200 {
		return 10 * time.Second
	}

	if du := time.Since(startAt); du <= 24*time.Hour {
		return 30 * time.Second
	}

	return time.Minute
}
