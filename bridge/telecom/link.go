package telecom

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
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
	// Fetch(context.Context, opcode.URLer, io.Reader, http.Header) (*http.Response, error)
	// Oneway(context.Context, opcode.URLer, io.Reader, http.Header) error
	// JSON(context.Context, opcode.URLer, any, any) error
	// OnewayJSON(context.Context, opcode.URLer, any) error
}

func Dial(parent context.Context, hide Hide, slog logback.Logger) (Linker, error) {
	addrs := formatAddrs(hide.Servers)
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

func formatAddrs(addrs []string) Addresses {
	ret := make(Addresses, 0, len(addrs))
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		if addr == "" {
			continue
		}

		host, port := splitHostPort(addr)
		if port != "" {
			dest := net.JoinHostPort(host, port)
			ret = append(ret, &Address{TLS: true, Addr: dest, Name: host})
			ret = append(ret, &Address{Addr: dest, Name: host})
		} else {
			safe := net.JoinHostPort(host, "443")
			nosafe := net.JoinHostPort(host, "80")
			ret = append(ret, &Address{TLS: true, Addr: safe, Name: host})
			ret = append(ret, &Address{Addr: nosafe, Name: host})
		}
	}

	return ret
}

func splitHostPort(addr string) (string, string) {
	if u, err := url.Parse(addr); err == nil && u.Host != "" {
		addr = u.Host
	}

	if host, port, err := net.SplitHostPort(addr); err == nil {
		return host, port
	}

	return addr, ""
}
