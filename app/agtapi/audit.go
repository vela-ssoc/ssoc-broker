package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/audit"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/xgfone/ship/v5"
)

func Audit(auditor audit.Auditor) route.Router {
	return &auditREST{
		auditor: auditor,
	}
}

type auditREST struct {
	auditor audit.Auditor
}

func (rest *auditREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/audit/risk").POST(rest.Risk)
	r.Route("/broker/audit/event").POST(rest.Event)
}

func (rest *auditREST) Risk(c *ship.Context) error {
	var req model.Risk
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	req.Inet = inf.Inet().String()
	req.MinionID = inf.Issue().ID

	return rest.auditor.Risk(ctx, &req)
}

func (rest *auditREST) Event(c *ship.Context) error {
	var req model.Event
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	req.Inet = inf.Inet().String()
	req.MinionID = inf.Issue().ID

	return rest.auditor.Event(ctx, &req)
}
