package restapi

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type Tunnel struct {
	v1 *websocket.Upgrader
	v3 *websocket.Upgrader
}

func NewTunnel() *Tunnel {
	return &Tunnel{
		v1: &websocket.Upgrader{
			HandshakeTimeout: 30 * time.Second,
		},
		v3: &websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
			Subprotocols:     []string{"smux", "yamux", "vela"},
		},
	}
}

func (tnl *Tunnel) RegisterHTTP(g shipx.RouteGroup) error {
	g.ExposeV1.Group.Route("/tunnel").GET(tnl.openV1)
	g.ExposeV3.Group.Route("/tunnel").GET(tnl.openV3)

	return nil
}

func (tnl *Tunnel) openV1(c *ship.Context) error {
	return nil
}

func (tnl *Tunnel) openV3(c *ship.Context) error {
	return nil
}
