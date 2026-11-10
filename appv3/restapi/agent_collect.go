package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	"github.com/xgfone/ship/v5"
)

type AgentCollect struct {
	svc *service.AgentCollect
}

func NewAgentCollect(svc *service.AgentCollect) *AgentCollect {
	return &AgentCollect{svc: svc}
}

func (ac *AgentCollect) RegisterHTTP(g shipx.RouteGroup) error {
	g.AgentV1.Group.Route("/broker/collect/agent/sysinfo").POST(ac.sysinfo)

	return nil
}

func (ac *AgentCollect) sysinfo(c *ship.Context) error {
	var param request.MinionSysinfo
	if err := c.Bind(&param); err != nil {
		return err
	}

	ctx := c.Request().Context()
	node := muxproto.FromContext(ctx)

	return ac.svc.Sysinfo(ctx, node, param)
}
