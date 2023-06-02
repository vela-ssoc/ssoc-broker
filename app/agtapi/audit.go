package agtapi

import (
	"time"

	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/xgfone/ship/v5"
)

func Audit() route.Router {
	return &auditREST{}
}

type auditREST struct{}

func (rest *auditREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/audit/risk").POST(rest.Risk)
	r.Route("/broker/audit/event").POST(rest.Event)
}

func (rest *auditREST) Risk(c *ship.Context) error {
	return nil
}

func (rest *auditREST) Event(c *ship.Context) error {
	var req model.Event
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	req.ID = 0
	req.Inet = inf.Inet().String()
	req.MinionID = inf.Issue().ID
	req.OccurAt = time.Now()

	_ = query.Event.WithContext(ctx).Create(&req)

	return nil
}
