package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb-itai/integration/elastic"
	"github.com/xgfone/ship/v5"
)

func Elastic(esc elastic.Searcher) route.Router {
	return &elasticREST{esc: esc}
}

type elasticREST struct {
	esc elastic.Searcher
}

func (rest *elasticREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/proxy/elastic").Data(route.Ignore()).Any(rest.Elastic)
	r.Route("/broker/proxy/elastic/*path").Data(route.Ignore()).Any(rest.Elastic)
}

func (rest *elasticREST) Elastic(c *ship.Context) error {
	path := "/" + c.Param("path")
	w, r := c.Response(), c.Request()
	ctx := r.Context()
	r.URL.Path = path
	r.RequestURI = path
	err := rest.esc.ServeHTTP(ctx, w, r)
	if err != nil {
		c.Warnf("es 代理执行错误：%s", err)
	}

	return err
}
