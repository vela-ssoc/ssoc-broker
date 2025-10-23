package serverd

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/netip"
	"time"

	"github.com/vela-ssoc/ssoc-broker/channel/linkhub"
)

func defaultValid(v any) error {
	req, ok := v.(authRequest)
	if !ok {
		return fmt.Errorf("认证报文结构体无效")
	}
	if req.MachineID == "" {
		return errors.New("machine_id 必须填写")
	}
	if _, err := netip.ParseAddr(req.Inet); err != nil {
		return errors.New("IP 填写错误")
	}

	return nil
}

type Optioner interface {
	options() []func(before option) (after option)
}

type Limiter interface {
	Allowed() bool
}

type unlimited struct{}

func (*unlimited) Allowed() bool { return true }

type option struct {
	logger  *slog.Logger
	valid   func(any) error
	server  *http.Server
	limit   Limiter
	huber   linkhub.Huber
	timeout time.Duration
}

func NewOption() OptionBuilder {
	return OptionBuilder{}
}

type OptionBuilder struct {
	opts []func(option) option
}

func (ob OptionBuilder) options() []func(option) option {
	return ob.opts
}

func (ob OptionBuilder) Logger(v *slog.Logger) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		o.logger = v
		return o
	})
	return ob
}

func (ob OptionBuilder) Valid(v func(any) error) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		o.valid = v
		return o
	})
	return ob
}

func (ob OptionBuilder) Server(v *http.Server) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		o.server = v
		return o
	})
	return ob
}

func (ob OptionBuilder) Handler(v http.Handler) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		if o.server == nil {
			o.server = new(http.Server)
		}
		o.server.Handler = v

		return o
	})
	return ob
}

func (ob OptionBuilder) Limit(v Limiter) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		o.limit = v
		return o
	})
	return ob
}

func (ob OptionBuilder) Timeout(v time.Duration) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		o.timeout = v
		return o
	})
	return ob
}

func (ob OptionBuilder) Huber(v linkhub.Huber) OptionBuilder {
	ob.opts = append(ob.opts, func(o option) option {
		o.huber = v
		return o
	})
	return ob
}

func defaultOption(o option) option {
	if o.valid == nil {
		o.valid = defaultValid
	}
	if o.server == nil {
		o.server = new(http.Server)
	}
	if o.server.Handler == nil {
		o.server.Handler = http.NotFoundHandler()
	}
	if o.limit == nil {
		o.limit = new(unlimited)
	}
	if o.huber == nil {
		o.huber = linkhub.NewSafeMap()
	}
	if o.timeout <= 0 {
		o.timeout = 30 * time.Second
	}

	return o
}
