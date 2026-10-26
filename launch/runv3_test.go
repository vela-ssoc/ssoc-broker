package launch_test

import (
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common-mb/param/negotiate"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"gorm.io/gorm"
)

type v3Input struct {
	route shipx.RouteGroup
	issue negotiate.Issue
	db    *gorm.DB
	mux   brokclient.Muxer
	hide  config.BootConfig
	cron  *cronv3.Crontab
	log   *slog.Logger
}
