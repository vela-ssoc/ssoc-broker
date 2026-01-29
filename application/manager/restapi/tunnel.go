package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/application/manager/request"
	"github.com/vela-ssoc/ssoc-broker/application/manager/service"
	"github.com/xgfone/ship/v5"
)

type Tunnel struct {
	svc *service.Tunnel
}

func NewTunnel(svc *service.Tunnel) *Tunnel {
	return &Tunnel{
		svc: svc,
	}
}

func (tnl *Tunnel) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/tunnel/stat").GET(tnl.stat)
	rgb.Route("/tunnel/limit").POST(tnl.limit)

	return nil
}

func (tnl *Tunnel) stat(c *ship.Context) error {
	ret := tnl.svc.Stat()

	return c.JSON(http.StatusOK, ret)
}

func (tnl *Tunnel) limit(c *ship.Context) error {
	req := new(request.TunnelLimit)
	if err := c.Bind(req); err != nil {
		return err
	}
	tnl.svc.Limit(req)

	return c.NoContent(http.StatusNoContent)
}
