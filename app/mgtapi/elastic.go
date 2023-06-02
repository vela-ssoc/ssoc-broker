package mgtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/xgfone/ship/v5"
)

func Elastic(cfg elastic.Configurer) route.Router {
	return &elasticREST{
		cfg: cfg,
	}
}

type elasticREST struct {
	cfg elastic.Configurer
}

func (rest *elasticREST) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathElasticReset).POST(rest.Reset)
}

// Reset 重置
func (rest *elasticREST) Reset(c *ship.Context) error {
	rest.cfg.Reset()
	c.Infof("elastic reset()")
	return nil
}
