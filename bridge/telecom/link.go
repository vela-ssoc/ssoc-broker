package telecom

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/param/negotiate"
	"github.com/vela-ssoc/vela-common-mba/netutil"
)

var ErrEmptyAddress = errors.New("服务端地址不能为空")

type Linker interface {
	Hide() negotiate.Hide
	Ident() negotiate.Ident
	Issue() negotiate.Issue
	Name() string
	JoinAt() time.Time
	Listen() net.Listener
	Reconnect(context.Context) error
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

func Dial(parent context.Context, hide *negotiate.Hide, version string, log *slog.Logger) (Linker, error) {
	addrs := hide.Servers.Preformat()
	if len(addrs) == 0 {
		return nil, ErrEmptyAddress
	}

	dialer := newIterDial(addrs)
	if version == "" {
		version = "0.0.0"
	}
	bc := &brokerClient{
		hide:    *hide,
		version: version,
		log:     log,
		dialer:  dialer,
	}
	trip := &http.Transport{DialContext: bc.dialContext}
	bc.client = netutil.NewClient(trip)

	if err := bc.dial(parent); err != nil {
		return nil, err
	}

	go bc.heartbeat(time.Minute)

	return bc, nil
}
