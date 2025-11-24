package clientd

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

func NewOption() OptionBuilder {
	return OptionBuilder{}
}

type OptionBuilder struct {
	opts []func(before option) (after option)
}

func (ob OptionBuilder) List() []func(option) option {
	return ob.opts
}

func (ob OptionBuilder) Logger(v *slog.Logger) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		o.logger = v
		return o
	})
	return ob
}

func (ob OptionBuilder) Handler(v http.Handler) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		o.handler = v
		return o
	})
	return ob
}

type option struct {
	handler http.Handler
	logger  *slog.Logger
	dialer  *websocket.Dialer
	timeout time.Duration
}

func fallbackOption() OptionBuilder {
	return OptionBuilder{
		opts: []func(option) option{
			func(o option) option {
				if o.handler == nil {
					o.handler = http.NotFoundHandler()
				}
				if o.timeout <= 0 {
					o.timeout = 30 * time.Second
				}
				if o.dialer == nil {
					o.dialer = &websocket.Dialer{
						Proxy:            http.ProxyFromEnvironment,
						TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
						HandshakeTimeout: o.timeout,
					}
				}

				return o
			},
		},
	}
}
