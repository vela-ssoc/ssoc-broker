package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/xgfone/ship/v5"
)

func Heart() route.Router {
	return &heartREST{}
}

type heartREST struct{}

func (rest *heartREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/minion/ping").POST(rest.Ping)
}

func (rest *heartREST) Ping(c *ship.Context) error {
	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	c.Debugf("%s(%d) 发来了心跳", inf.Inet(), inf.Issue().ID)

	return nil
}
