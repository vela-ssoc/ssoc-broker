package agtapi

import (
	"github.com/vela-ssoc/ssoc-broker/app/agtsvc"
	"github.com/vela-ssoc/ssoc-broker/app/internal/param"
	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
	"github.com/xgfone/ship/v5"
)

func NewAgentEmergencySnapshot(svc *agtsvc.AgentEmergencySnapshot) *AgentEmergencySnapshot {
	return &AgentEmergencySnapshot{
		svc: svc,
	}
}

type AgentEmergencySnapshot struct {
	svc *agtsvc.AgentEmergencySnapshot
}

func (aes *AgentEmergencySnapshot) Route(r *ship.RouteGroupBuilder) {
	r.Route("/agent-emergency-snapshot/report").POST(aes.report)
}

func (aes *AgentEmergencySnapshot) report(c *ship.Context) error {
	req := new(param.AgentEmergencySnapshotReport)
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	return aes.svc.Report(ctx, req, inf)
}
