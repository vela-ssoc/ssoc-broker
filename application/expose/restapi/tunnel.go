package restapi

import (
	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/xgfone/ship/v5"
)

type Tunnel struct {
	acpt muxserver.MUXAccepter
}

func NewTunnel(acpt muxserver.MUXAccepter) *Tunnel {
	return &Tunnel{
		acpt: acpt,
	}
}

func (tnl *Tunnel) RegisterRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/minion").CONNECT(tnl.legacy)
	rgb.Route("/tunnel").GET(tnl.open)

	return nil
}

// legacy 旧版 agent 上线接入端点。
// 兼容用，随着迭代此接口将会移除。
//
// Deprecated: use open.
func (tnl *Tunnel) legacy(c *ship.Context) error {
	w, r := c.ResponseWriter(), c.Request()
	_ = tnl.acpt.AcceptTCP(w, r)

	return nil
}

func (tnl *Tunnel) open(c *ship.Context) error {
	return nil
}
