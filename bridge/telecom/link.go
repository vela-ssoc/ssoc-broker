package telecom

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mba/netutil"
)

var ErrEmptyAddress = errors.New("服务端地址不能为空")

type Linker interface {
	Hide() Hide
	Ident() Ident
	Issue() Issue
	Name() string
	JoinAt() time.Time
	Listen() net.Listener
	Reconnect(context.Context) error
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	// Fetch(context.Context, opcode.URLer, io.Reader, http.Header) (*http.Response, error)
	// Oneway(context.Context, opcode.URLer, io.Reader, http.Header) error
	// JSON(context.Context, opcode.URLer, any, any) error
	// OnewayJSON(context.Context, opcode.URLer, any) error
}

func Dial(parent context.Context, hide Hide, slog logback.Logger) (Linker, error) {
	addrs := hide.Servers.Preformat()
	if len(addrs) == 0 {
		return nil, ErrEmptyAddress
	}

	dialer := newIterDial(addrs)
	bc := &brokerClient{
		hide:   hide,
		slog:   slog,
		dialer: dialer,
	}
	trip := &http.Transport{DialContext: bc.dialContext}
	bc.client = netutil.NewClient(trip)

	if err := bc.dial(parent); err != nil {
		return nil, err
	}

	// go bc.heartbeat(time.Minute)

	return bc, nil
}
