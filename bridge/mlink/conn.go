package mlink

import (
	"context"
	"net"

	"github.com/vela-ssoc/ssoc-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-common-mba/smux"
)

type Infer interface {
	Ident() gateway.Ident
	Issue() gateway.Issue
	Inet() net.IP
}

type connect struct {
	id    int64
	ident gateway.Ident
	issue gateway.Issue
	mux   *smux.Session
	// mux   spdy.Muxer
}

func (c *connect) Ident() gateway.Ident { return c.ident }
func (c *connect) Issue() gateway.Issue { return c.issue }
func (c *connect) Inet() net.IP         { return c.ident.Inet }

type contextKey struct{ name string }

var minionCtxKey = &contextKey{name: "minion-context"}

func Ctx(ctx context.Context) Infer {
	if ctx != nil {
		infer, _ := ctx.Value(minionCtxKey).(Infer)
		return infer
	}

	return nil
}
