package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/integration/alarm"
	"github.com/xgfone/ship/v5"
)

func Audit(alert alarm.Alerter) route.Router {
	return &auditREST{
		alert: alert,
	}
}

type auditREST struct {
	alert alarm.Alerter
}

func (rest *auditREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/audit/risk").POST(rest.Risk)
	r.Route("/broker/audit/event").POST(rest.Event)
}

func (rest *auditREST) Risk(c *ship.Context) error {
	var req param.AuditRiskRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid := inf.Issue().ID
	inet := inf.Inet().String()

	rsk := req.Model(mid, inet)

	return rest.alert.RiskSaveAndAlert(ctx, rsk)
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

	return rest.alert.EventSaveAndAlert(ctx, &req)
}
