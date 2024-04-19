package mgtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb-itai/accord"
	"github.com/vela-ssoc/vela-common-mb-itai/integration/dong"
	"github.com/vela-ssoc/vela-common-mb-itai/integration/elastic"
	"github.com/vela-ssoc/vela-common-mb-itai/integration/ntfmatch"
	"github.com/vela-ssoc/vela-common-mb-itai/storage/v2"
	"github.com/xgfone/ship/v5"
)

func Reset(
	store storage.Storer,
	esCfg elastic.Configurer,
	ntf ntfmatch.Matcher,
	emc dong.Configurer,
) route.Router {
	return &resetREST{
		store: store,
		esCfg: esCfg,
		ntf:   ntf,
		emc:   emc,
	}
}

type resetREST struct {
	esCfg elastic.Configurer
	ntf   ntfmatch.Matcher
	store storage.Storer
	emc   dong.Configurer
}

func (rest *resetREST) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathElasticReset).
		Data(route.Named("elastic 配置 reset")).POST(rest.Elastic)
	r.Route(accord.PathStoreReset).
		Data(route.Named("store 配置 reset")).POST(rest.Store)
	r.Route(accord.PathNotifierReset).
		Data(route.Named("告警人 reset")).POST(rest.Notifier)
	r.Route(accord.PathEmcReset).
		Data(route.Named("咚咚服务号 reset")).POST(rest.Emc)
	r.Route(accord.PathEmailReset).
		Data(route.Named("邮箱发送账号 reset")).POST(rest.Email)
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
	rest.store.Forget(id)

	return nil
}

func (rest *resetREST) Notifier(c *ship.Context) error {
	c.Infof("告警人 reset")
	rest.ntf.Reset()
	return nil
}

func (rest *resetREST) Emc(c *ship.Context) error {
	c.Infof("咚咚服务号 reset")
	rest.emc.Forget()
	return nil
}

func (rest *resetREST) Email(c *ship.Context) error {
	c.Infof("邮箱发送账号 reset")
	return nil
}
