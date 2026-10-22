package launch

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/NVIDIA/gontainer/v2"
	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/ssoc-broker/appv3/restapi"
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/cronjob/cronfunc"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/repository"
	"github.com/vela-ssoc/ssoc-common-mb/dbconn"
	"github.com/vela-ssoc/ssoc-common-mb/param/negotiate"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"gorm.io/gorm"
)

type v3Input struct {
	route shipx.RouteGroup
	issue negotiate.Issue
	db    *gorm.DB
	mux   brokclient.Muxer
	hide  config.HideConfig
	cron  *cronv3.Crontab
	log   *slog.Logger
}

func runV3(ctx context.Context, input v3Input) error {
	wsu := &websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool {
			return true
		},
	}

	bootcfg := input.mux.Config()
	gopts := []gontainer.Option{
		gontainer.NewService(wsu),
		gontainer.NewService(bootcfg),
		gontainer.NewService(input.hide),
		gontainer.NewService(input.db),
		gontainer.NewService(input.mux),
		gontainer.NewService(input.issue),
		gontainer.NewService(input.log),
		gontainer.NewService(input.cron),
		gontainer.NewService(dbconn.MongoDBConfig{URI: bootcfg.Database.URI}),

		gontainer.NewFactory(dbconn.OpenMongoDB),
		gontainer.NewFactory(repository.NewBaseDB),
		gontainer.NewFactory(polydb.NewSource),

		gontainer.NewFactory(restapi.NewExit),
		gontainer.NewFactory(restapi.NewConfig),
		gontainer.NewFactory(restapi.NewPing),
		gontainer.NewFactory(restapi.NewPprof),

		gontainer.NewFactory(service.NewReverseProxy),
		gontainer.NewFactory(restapi.NewReverseProxy),

		gontainer.NewFactory(restapi.NewRoute),
		gontainer.NewFactory(restapi.NewPTY),
		gontainer.NewService(restapi.NewWebDAV("/")),

		gontainer.NewFactory(service.NewZipArchive),
		gontainer.NewFactory(restapi.NewZipArchive),

		gontainer.NewFactory(cronfunc.NewTunnel),
		gontainer.NewEntrypoint(func(tab *cronv3.Crontab, tasks gontainer.Multiple[cronv3.Tasker]) error {
			for _, task := range tasks {
				if err := tab.AddTask(task); err != nil {
					return err
				}
			}

			return nil
		}),

		gontainer.NewEntrypoint(input.route.RegisterHTTP),
	}
	if err := gontainer.Run(gopts...); err != nil {
		return err
	}

	return nil
}
