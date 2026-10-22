package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/xgfone/ship/v5"
)

type Config struct {
	hc config.HideConfig
	bc model.BrokerConfig
}

func NewConfig(hide config.HideConfig, boot model.BrokerConfig) *Config {
	return &Config{hc: hide, bc: boot}
}

func (cf *Config) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Route("/config/boot").GET(cf.boot)
	g.ManagerV3.Route("/config/hide").GET(cf.hide)

	return nil
}

func (cf *Config) boot(c *ship.Context) error {
	return c.JSON(http.StatusOK, cf.bc)
}

func (cf *Config) hide(c *ship.Context) error {
	return c.JSON(http.StatusOK, cf.hc)
}
