package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	"github.com/xgfone/ship/v5"
)

type Minion struct {
	svc *service.Minion
}

func NewMinion(svc *service.Minion) *Minion {
	return &Minion{svc: svc}
}

func (p *Minion) RegisterHTTP(g shipx.RouteGroup) error {
	g.AgentV1.Group.Route("/minion/ping").POST(p.ping)

	return nil
}

func (p *Minion) ping(c *ship.Context) error {
	ctx := c.Request().Context()
	node := muxproto.FromContext(ctx)

	return p.svc.Ping(ctx, node)
}
