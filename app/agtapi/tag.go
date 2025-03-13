package agtapi

import (
	"github.com/vela-ssoc/ssoc-broker/app/agtsvc"
	"github.com/vela-ssoc/ssoc-broker/app/internal/param"
	"github.com/vela-ssoc/ssoc-broker/app/route"
	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
	"github.com/xgfone/ship/v5"
)

func Tag(svc agtsvc.TagService) route.Router {
	return &tagREST{
		svc: svc,
	}
}

type tagREST struct {
	svc agtsvc.TagService
}

func (rest *tagREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/operate/tag").POST(rest.Tag)
}

func (rest *tagREST) Tag(c *ship.Context) error {
	var req param.TagRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid := inf.Issue().ID
	if req.Empty() {
		return nil
	}

	return rest.svc.Update(ctx, mid, req.Add, req.Del)
}
