package mgtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/vela-ssoc/vela-common-mb/integration/ntfmatch"
	"github.com/vela-ssoc/vela-common-mb/storage"
	"github.com/xgfone/ship/v5"
)

func Reset(
	store storage.Storer,
	esCfg elastic.Configurer,
	ntf ntfmatch.Matcher,
) route.Router {
	return &resetREST{
		store: store,
		esCfg: esCfg,
		ntf:   ntf,
	}
}

type resetREST struct {
	esCfg elastic.Configurer
	ntf   ntfmatch.Matcher
	store storage.Storer
}

func (rest *resetREST) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathElasticReset).
		Data(route.Named("elastic 配置 reset")).POST(rest.Elastic)
	r.Route(accord.PathStoreReset).
		Data(route.Named("store 配置 reset")).POST(rest.Store)
	r.Route(accord.PathNotifierReset).
		Data(route.Named("告警人 reset")).POST(rest.Store)
}

func (rest *resetREST) Elastic(c *ship.Context) error {
	rest.esCfg.Reset()
	c.Infof("elastic 配置 reset")
	return nil
}

func (rest *resetREST) Store(c *ship.Context) error {
	var req accord.StoreRestRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	id := req.ID
	c.Infof("store reset: %s", id)
	rest.store.Reset(id)

	return nil
}

func (rest *resetREST) Notifier(c *ship.Context) error {
	c.Infof("store reset: %s")
	rest.ntf.Reset()
	return nil
}
