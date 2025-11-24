package clientd

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/options"
	"github.com/vela-ssoc/ssoc-common/linkhub"
	"github.com/xtaci/smux"
)

func Open(ctx context.Context, cfg Config, opts ...options.Lister[option]) (Muxer, Database, error) {
	if err := cfg.preparse(); err != nil {
		return nil, Database{}, err
	}

	opts = append(opts, fallbackOption())
	opt := options.Eval(opts...)

	mux := new(safeMuxer)
	bc := &brokerClient{
		cfg: cfg,
		opt: opt,
		mux: mux,
		ctx: ctx,
	}
	dbc, err := bc.open()
	if err != nil {
		return nil, dbc, err
	}

	ln := &muxerListener{mux: mux}
	go bc.serve(ln)

	return mux, dbc, nil
}

type brokerClient struct {
	cfg Config
	opt option
	mux *safeMuxer
	ctx context.Context
}

// open 持续尝试连接 manager 直至成功或错误。
func (bc *brokerClient) open() (Database, error) {
	addrs := bc.cfg.Addresses
	timeout := bc.opt.timeout
	bc.log().Info("准备建立连接通道", "addresses", addrs)

	beginAt := time.Now()
	var fails int
	for {
		attrs := []any{
			slog.Any("addresses", addrs),
			slog.Duration("timeout", timeout),
		}
		sess, resp, err := bc.connects(addrs, timeout)
		if err == nil {
			bc.mux.store(sess)
			bc.log().Info("broker 通道连接认证成功", attrs...)
			return resp.Database, nil
		}

		fails++
		du := bc.waitN(fails, beginAt)
		attrs = append(attrs, slog.Time("begin_at", beginAt))
		attrs = append(attrs, slog.Int("fails", fails))
		attrs = append(attrs, slog.Duration("timeout", timeout))
		attrs = append(attrs, slog.Any("error", err))
		bc.log().Warn("broker 通道连接或认证失败，稍后重试", attrs...)
		if err = bc.sleep(du); err != nil {
			attrs = append(attrs, slog.Any("sleep_error", err))
			bc.log().Error("context 取消，退出重连机制", attrs...)
			return Database{}, err
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
	sess, err := bc.dial(addr, timeout)
	if err != nil {
		return nil, nil, err
	}
	resp, err := bc.authentication(sess, timeout)
	if err != nil {
		_ = sess.Close()
		return nil, nil, err
	}
	if err = resp.checkError(); err != nil {
		_ = sess.Close()
		return nil, nil, err
	}

	return sess, resp, nil
}

func (bc *brokerClient) dial(addr string, timeout time.Duration) (*smux.Session, error) {
	ctx, cancel := context.WithTimeout(bc.ctx, timeout)
	defer cancel()

	dialer := bc.opt.dialer
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
		return nil, err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer pre.Close()
	_ = pre.SetDeadline(time.Now().Add(timeout))

	req := &authRequest{Secret: bc.cfg.Secret, Semver: bc.cfg.Semver}
	if err = linkhub.WriteAuth(pre, req); err != nil {
		return nil, err
	}

	resp := new(authResponse)
	if err = linkhub.ReadAuth(pre, resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (bc *brokerClient) serve(ln net.Listener) {
	const sleep = 3 * time.Second

	for {
		han := bc.opt.handler
		srv := &http.Server{Handler: han}
		err := srv.Serve(ln)
		_ = ln.Close()

		attrs := []any{slog.Any("error", err), slog.Duration("timeout", sleep)}
		bc.log().Warn("broker 掉线了", attrs...)
		_ = bc.sleep(sleep)
		if _, err = bc.open(); err != nil {
			break
		}
	}
}

func (bc *brokerClient) log() *slog.Logger {
	if l := bc.opt.logger; l != nil {
		return l
	}

	return slog.Default()
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
