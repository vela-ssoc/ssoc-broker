package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type ReverseProxy struct {
	svc *service.ReverseProxy
}

func NewReverseProxy(svc *service.ReverseProxy) *ReverseProxy {
	return &ReverseProxy{svc: svc}
}

func (rp *ReverseProxy) RegisterHTTP(g shipx.RouteGroup) error {
	g.AgentV3.Route("/reverse/:location").Any(rp.handle)
	g.AgentV3.Route("/reverse/:location/*path").Any(rp.handle)

	g.ManagerV3.Route("/reverse-proxy").POST(rp.forget)

	return nil
}

func (rp *ReverseProxy) handle(c *ship.Context) error {
	w, r := shipx.Strip(c, "path")
	ctx := r.Context()
	loc := c.Param("location")

	if err := rp.svc.Handle(ctx, loc, w, r); err != nil {
		return ship.ErrNotFound
	}

	return nil
}

func (rp *ReverseProxy) forget(c *ship.Context) error {
	rp.svc.Forget(nil)
	return nil
}
