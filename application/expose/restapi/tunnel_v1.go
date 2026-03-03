package restapi

import (
	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/xgfone/ship/v5"
)

type TunnelV1 struct {
	acpt muxserver.MUXAccepter
}

func NewTunnelV1(acpt muxserver.MUXAccepter) *TunnelV1 {
	return &TunnelV1{
		acpt: acpt,
	}
}

func (tnl *TunnelV1) RegisterRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/minion").CONNECT(tnl.legacy)

	return nil
}

// legacy 旧版 agent 上线接入端点。
// 兼容用，随着迭代此接口将会移除。
//
// Deprecated: use open.
func (tnl *TunnelV1) legacy(c *ship.Context) error {
	w, r := c.ResponseWriter(), c.Request()
	_ = tnl.acpt.AcceptTCP(w, r)

	return nil
}
