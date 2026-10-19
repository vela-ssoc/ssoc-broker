package launch

import (
	"context"
	"log/slog"

	"github.com/NVIDIA/gontainer/v2"
	"github.com/vela-ssoc/ssoc-broker/appv3/restapi"
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/repository"
	"github.com/vela-ssoc/ssoc-common-mb/dbconn"
	"github.com/vela-ssoc/ssoc-common-mb/param/negotiate"
	"github.com/vela-ssoc/ssoc-common/banner"
)

func runV3(ctx context.Context, route shipx.RouteGroup, iss negotiate.Issue, log *slog.Logger) error {
	gopts := []gontainer.Option{
		gontainer.NewService(iss),
		gontainer.NewService(log),
		gontainer.NewService(dbconn.MongoDBConfig{URI: iss.Database.URI}),

		gontainer.NewFactory(dbconn.OpenMongoDB),
		gontainer.NewFactory(repository.NewBaseDB),

		gontainer.NewFactory(service.NewReverseProxy),
		gontainer.NewFactory(restapi.NewReverseProxy),

		gontainer.NewFactory(service.NewZipArchive),
		gontainer.NewFactory(restapi.NewZipArchive),

		gontainer.NewEntrypoint(route.RegisterHTTP),
	}
	if err := gontainer.Run(gopts...); err != nil {
		return err
	}

	return nil
}

//goland:noinspection GoUnhandledErrorResult
func RunV3(ctx context.Context, pcfg config.Parser) error {
	var bootcfg config.BootConfig
	if err := pcfg.Parse(ctx, &bootcfg); err != nil {
		return err
	}

	version := banner.Version()
	cliopts := brokclient.Options{
		Addresses: bootcfg.Addresses,
		Secret:    bootcfg.Secret,
		Version:   version,
	}
	mux, err := brokclient.Open(ctx, cliopts) // 连接中心端直至成功。
	if err != nil {
		return err
	}
	defer mux.Close()

	muxcfg := mux.Config()
	mdb, err := dbconn.OpenMongoDB(dbconn.MongoDBConfig{URI: muxcfg.URI})
	if err != nil {
		return err
	}
	// 开始上线
	_ = mdb

	return nil
}
