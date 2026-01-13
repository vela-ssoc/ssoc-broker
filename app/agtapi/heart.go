package agtapi

import (
	"time"

	"github.com/vela-ssoc/ssoc-broker/app/route"
	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/xgfone/ship/v5"
)

func Heart(qry *query.Query) route.Router {
	return &heartREST{
		qry: qry,
	}
}

type heartREST struct {
	qry *query.Query
}

func (rest *heartREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/minion/ping").Data(route.Ignore()).POST(rest.Ping)
}

func (rest *heartREST) Ping(c *ship.Context) error {
	now := time.Now()
	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	id := inf.Issue().ID
	c.Debugf("%s(%d) 发来了心跳", inf.Inet(), id)
	tbl := rest.qry.Minion
	dao := tbl.WithContext(ctx)

	_, err := dao.Where(tbl.ID.Eq(id)).
		UpdateSimple(tbl.HeartbeatAt.Value(now))

	return err
}
