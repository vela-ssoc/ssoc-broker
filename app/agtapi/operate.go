package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/xgfone/ship/v5"
)

func Operate() route.Router {
	return &operateREST{}
}

type operateREST struct{}

func (rest *operateREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/operate/tag").POST(rest.Tag)
}

func (rest *operateREST) Tag(c *ship.Context) error {
	return nil
}
