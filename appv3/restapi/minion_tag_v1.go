package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	"github.com/xgfone/ship/v5"
)

type MinionTagV1 struct {
	svc *service.MinionTagV1
}

func NewMinionTagV1(svc *service.MinionTagV1) *MinionTagV1 {
	return &MinionTagV1{svc: svc}
}

func (p *MinionTagV1) RegisterHTTP(g shipx.RouteGroup) error {
	g.AgentV1.Group.Route("/broker/operate/tag").POST(p.tag)

	return nil
}

func (p *MinionTagV1) tag(c *ship.Context) error {
	var param request.MinionTag
	if err := c.Bind(&param); err != nil {
		return err
	}
	if param.IsZero() {
		return nil
	}

	ctx := c.Request().Context()
	node := muxproto.FromContext(ctx)

	return p.svc.Update(ctx, node, param.Add, param.Del)
}
