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
	rgb.Route("/tunnel").GET(tnl.open)

	return nil
}

func (tnl *Tunnel) open(c *ship.Context) error {
	return nil
}
