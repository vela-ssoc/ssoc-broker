package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/response"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-common-mb/bininfo"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/xgfone/ship/v5"
)

type Itself struct {
	boot config.BootConfig
	cfg  model.BrokerConfig
}

func NewItself(boot config.BootConfig, cfg model.BrokerConfig) *Itself {
	return &Itself{
		boot: boot,
		cfg:  cfg,
	}
}

func (cf *Itself) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/itself/config").GET(cf.config)
	g.ManagerV3.Group.Route("/itself/buildinfo").GET(cf.buildinfo)

	return nil
}

func (cf *Itself) config(c *ship.Context) error {
	ret := &response.ItselfConfig{Boot: cf.boot, Config: cf.cfg}
	return c.JSON(http.StatusOK, ret)
}

func (cf *Itself) buildinfo(c *ship.Context) error {
	ret := bininfo.Self()
	return c.JSON(http.StatusOK, ret)
}
