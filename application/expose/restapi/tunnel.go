package restapi

import (
	"net/http"

	"github.com/xgfone/ship/v5"
)

func NewTunnel(upg http.Handler) *Tunnel {
	return &Tunnel{
		upg: upg,
	}
}

type Tunnel struct {
	upg http.Handler
}

func (tnl *Tunnel) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/tunnel").GET(tnl.open)

	return nil
}

// open agent 的接入点。
func (tnl *Tunnel) open(c *ship.Context) error {
	w, r := c.Response(), c.Request()
	tnl.upg.ServeHTTP(w, r)
	return nil
}
