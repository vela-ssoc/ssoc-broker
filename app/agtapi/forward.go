package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/xgfone/ship/v5"
)

func Forward(esc elastic.Searcher) route.Router {
	return &forwardREST{
		esc: esc,
	}
}

type forwardREST struct {
	esc elastic.Searcher
}

func (rest *forwardREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/forward/elastic").POST(rest.Elastic)
}

func (rest *forwardREST) Elastic(c *ship.Context) error {
	r := c.Request()
	ctx := r.Context()
	if _, err := rest.esc.Bulk(ctx, r.Body); err != nil {
		c.Warnf("es bulk 接口错误：%s", err)
		return err
	}

	return nil
}
