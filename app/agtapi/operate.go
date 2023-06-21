package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/app/service"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/xgfone/ship/v5"
)

func Operate(svc service.OperateService) route.Router {
	return &operateREST{
		svc: svc,
	}
}

type operateREST struct {
	svc service.OperateService
}

func (rest *operateREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/operate/tag").POST(rest.Tag)
}

func (rest *operateREST) Tag(c *ship.Context) error {
	var req param.TagRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid := inf.Issue().ID

	return rest.svc.Update(ctx, mid, &req)
}
