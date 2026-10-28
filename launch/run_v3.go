package launch

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/NVIDIA/gontainer/v2"
	"github.com/gorilla/websocket"
	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-broker/appv3/restapi"
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/cronfunc"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common-mb/bininfo"
	"github.com/vela-ssoc/ssoc-common-mb/crontab"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/repository"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/varload"
	"github.com/vela-ssoc/ssoc-common-mb/dbconn"
	"github.com/vela-ssoc/ssoc-common-mb/tlscert"
	"github.com/vela-ssoc/ssoc-common-mb/validation"
	"github.com/vela-ssoc/ssoc-common-mb/vmexport"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/xgfone/ship/v5"
)

func RunV3(ctx context.Context, cfg string) error {
	ls := varload.NewLoaders(varload.JSONC(cfg), os.IsNotExist, steganoFile(os.Args[0]))
	var boot config.BootConfig
	if err := ls.Load(ctx, &boot); err != nil {
		return err
	}

	return Exec(ctx, boot)
}

//goland:noinspection GoUnhandledErrorResult
func Exec(ctx context.Context, boot config.BootConfig) error {
	valid := validation.New()
	if err := valid.RegisterCustomValidations(validation.All()); err != nil {
		return err
	}

	exposeMUX := ship.Default()
	managerMUX := ship.Default()
	agentMUX := ship.Default()

	selfinfo := bininfo.Self()
	brokopts := brokclient.Options{
		Addresses: brokclient.NewEndpoints(boot.Addresses...),
		Secret:    boot.Secret,
		Version:   selfinfo.Version,
		Handler:   managerMUX,
		Validator: valid.Validate,
	}
	mux, err := brokclient.Open(ctx, brokopts)
	if err != nil {
		return err
	}
	defer mux.Close()

	// 初始化日志
	cfg := mux.Config()
	var logHandles []slog.Handler
	{
		logcfg := cfg.Logger
		opts := &slog.HandlerOptions{AddSource: true, Level: logcfg.LevelVar()}
		if logcfg.Console {
			logHandles = append(logHandles, logger.NewTint(os.Stdout, opts))
		}
		if lumber := logcfg.Lumber(); lumber != nil {
			defer lumber.Close()
			logHandles = append(logHandles, slog.NewJSONHandler(lumber, opts))
		}
	}
	logHandler := slog.NewMultiHandler(logHandles...)
	log := slog.New(logHandler)

	cronparser := crontab.NewSecondOptional()
	crond := crontab.NewTable(log, cron.WithParser(cronparser))
	crond.Start()
	defer crond.Stop()

	// 连接 MongoDB
	// 连接关系型数据库
	dbcfg := cfg.Database
	mdb, err := dbconn.OpenMDB(dbcfg.URI)
	if err != nil {
		log.Error("连接 MongoDB 出错", "err", err)
		return err
	}
	defer closeMDB(mdb)
	log.Info("MongoDB 数据库连接成功")

	rdb, err := dbconn.OpenRDB(dbcfg.DSN)
	if err != nil {
		log.Error("连接关系型数据库出错", "err", err)
		return err
	}
	defer closeRDB(rdb)
	log.Info("关系数据库连接成功", "dialect", rdb.Dialector.Name())

	// 如果需要创建关系型数据库的索引、数据合并，在此处实现。
	qry := query.Use(rdb)
	repo := repository.NewBaseDB(mdb, log)
	pdb := polydb.NewSource(repo, qry) // 数据库聚合

	wsu := &websocket.Upgrader{
		CheckOrigin: func(*http.Request) bool {
			return true
		},
	}
	transport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        20,
		MaxIdleConnsPerHost: 2,
		MaxConnsPerHost:     20,
		IdleConnTimeout:     30 * time.Second,
	}

	route := shipx.RouteGroup{
		ExposeV1: shipx.RouteEndpoint{
			Path:  "/api/v1",
			Group: exposeMUX.Group("/api/v1"),
		},
		ExposeV3: shipx.RouteEndpoint{
			Path:  "/api/v3",
			Group: exposeMUX.Group("/api/v3"),
		},
		ManagerV1: shipx.RouteEndpoint{
			Path:  "/api/v1",
			Group: managerMUX.Group("/api/v1"),
		},
		ManagerV3: shipx.RouteEndpoint{
			Path:  "/api/v3",
			Group: managerMUX.Group("/api/v3"),
		},
		AgentV1: shipx.RouteEndpoint{
			Path:  "/api/v1",
			Group: agentMUX.Group("/api/v1"),
		},
		AgentV3: shipx.RouteEndpoint{
			Path:  "/api/v3",
			Group: agentMUX.Group("/api/v3"),
		},
	}

	diopts := []gontainer.Option{
		gontainer.NewService(log),
		gontainer.NewService(qry), // v1
		gontainer.NewService(pdb),
		gontainer.NewService(mux),
		gontainer.NewService(wsu),
		gontainer.NewService(boot),
		gontainer.NewService(cfg),
		gontainer.NewService(crond),
		gontainer.NewService(transport),

		gontainer.NewFactory(restapi.NewItself),
		gontainer.NewFactory(restapi.NewItselfCrontab),
		gontainer.NewFactory(restapi.NewItselfExit),
		gontainer.NewFactory(restapi.NewItselfRoute),
		gontainer.NewFactory(restapi.NewPing),
		gontainer.NewFactory(restapi.NewPprof),
		gontainer.NewFactory(service.NewVictoriaMetrics),
		gontainer.NewFactory(service.NewReverseProxy),
		gontainer.NewFactory(restapi.NewReverseProxy),
		gontainer.NewFactory(restapi.NewSystemProcess),
		gontainer.NewFactory(restapi.NewSystemPTY),
		gontainer.NewService(restapi.NewWebDAV("/")),
		gontainer.NewFactory(service.NewZipArchive),
		gontainer.NewFactory(restapi.NewZipArchive),

		gontainer.NewFactory(cronfunc.NewTunnelStat),
		gontainer.NewFactory(vmexport.NewPsutil),
		gontainer.NewFactory(cronfunc.NewVMetric),

		gontainer.NewEntrypoint(route.RegisterRoutes),
		gontainer.NewEntrypoint(registerCronJob),
	}
	if err = gontainer.Run(diopts...); err != nil {
		log.Error("依赖注入出错", "err", err)
		fmt.Println(err.Error())
		return err
	}
	log.Info("依赖注入执行完毕")

	cfgsrv := cfg.Server
	tlspool := tlscert.NewMatch(nil, log)
	srv := &http.Server{
		Addr:      cfgsrv.Addr,
		Handler:   exposeMUX,
		TLSConfig: &tls.Config{GetCertificate: tlspool.GetCertificate},
	}
	errch := make(chan error, 1)
	go startHTTPS(errch, srv)
	select {
	case <-ctx.Done():
		err = context.Cause(ctx)
	case err = <-errch:
	}

	log.Error("程序运行结束", "err", err)
	_ = closeHTTP(srv)

	return err
}
