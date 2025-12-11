package clientd

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

type Options struct {
	// Handler 提供给 manager 调用服务接口。
	Handler http.Handler

	// Logger 日志。
	Logger *slog.Logger

	// Timeout 分别控制连接的超时和握手读写超时时间。
	Timeout time.Duration
}

func (opt Options) logger() *slog.Logger {
	if log := opt.Logger; log != nil {
		return log
	}

	return slog.Default()
}

func (opt Options) handler() http.Handler {
	if h := opt.Handler; h != nil {
		return h
	}
	opt.logger().Warn("broker 客户端没有配置 http handler，会导致 manager ⇋ broker 无法业务调用")

	return http.NotFoundHandler()
}

func (opt Options) timeout() time.Duration {
	if timeout := opt.Timeout; timeout > 0 {
		return timeout
	}

	return 30 * time.Second
}

func (opt Options) dialer() *websocket.Dialer {
	return &websocket.Dialer{
		HandshakeTimeout: opt.timeout(),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
}
