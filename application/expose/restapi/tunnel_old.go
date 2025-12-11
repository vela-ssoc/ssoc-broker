package restapi

import "github.com/xgfone/ship/v5"

func NewTunnelOld() *TunnelOld {
	return &TunnelOld{}
}

type TunnelOld struct{}

func (tnl *TunnelOld) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/minion").CONNECT(tnl.open)

	return nil
}

// open agent 的接入点。
func (tnl *TunnelOld) open(c *ship.Context) error {
	c.Infof("tunnel old open")
	return nil
}
