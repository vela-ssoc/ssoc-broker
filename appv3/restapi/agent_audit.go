package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	"github.com/xgfone/ship/v5"
)

type AgentAudit struct {
	svc *service.AgentAudit
}

func NewAgentAudit(svc *service.AgentAudit) *AgentAudit {
	return &AgentAudit{svc: svc}
}

func (aa *AgentAudit) RegisterHTTP(g shipx.RouteGroup) error {
	g.AgentV1.Group.Route("/broker/audit/event").POST(aa.event)
	g.AgentV1.Group.Route("/broker/audit/risk").POST(aa.risk)

	return nil
}

func (aa *AgentAudit) event(c *ship.Context) error {
	var param request.AgentAuditEvent
	if err := c.Bind(&param); err != nil {
		return err
	}

	ctx := c.Request().Context()
	node := muxproto.FromContext(ctx)

	return aa.svc.Event(ctx, node, param)
}

func (aa *AgentAudit) risk(c *ship.Context) error {
	var param request.AgentAuditRisk
	if err := c.Bind(&param); err != nil {
		return err
	}

	ctx := c.Request().Context()
	node := muxproto.FromContext(ctx)

	return aa.svc.Risk(ctx, node, param)
}
