package launch

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	agentapi "github.com/vela-ssoc/ssoc-broker/application/agent/restapi"
	agentsvc "github.com/vela-ssoc/ssoc-broker/application/agent/service"
	exposeapi "github.com/vela-ssoc/ssoc-broker/application/expose/restapi"
	exposesvc "github.com/vela-ssoc/ssoc-broker/application/expose/service"
	managerapi "github.com/vela-ssoc/ssoc-broker/application/manager/restapi"
	"github.com/vela-ssoc/ssoc-broker/channel/agtrpc"
	"github.com/vela-ssoc/ssoc-broker/channel/clientd"
	"github.com/vela-ssoc/ssoc-broker/channel/serverd"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/library/pipelog"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common/gopool"
	"github.com/vela-ssoc/ssoc-common/httpkit"
	"github.com/vela-ssoc/ssoc-common/linkhub"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/profile"
	"github.com/vela-ssoc/ssoc-common/shipx"
	"github.com/vela-ssoc/ssoc-common/sqldb"
	"github.com/vela-ssoc/ssoc-common/validation"
	"github.com/xgfone/ship/v5"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Run2(ctx context.Context, cfile string) error {
	if cfile != "" {
		return Exec(ctx, profile.NewFile[config.Config](cfile))
	}

	// TODO 读取自身隐写的内容

	return nil
}

func Exec(ctx context.Context, pfl profile.Reader[config.Config]) error {
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelDebug)
	logOption := &slog.HandlerOptions{AddSource: true, Level: logLevel}
	tintHandler := logger.NewTint(os.Stdout, logOption)
	logHandler := logger.Multi(tintHandler)
	log := slog.New(logHandler)
	log.Info("日志组件初始化完毕")

	cfg, err := pfl.Read(ctx)
	if err != nil {
		log.Error("加载配置文件错误", "error", err)
		return err
	}

	valid := validation.New()
	if err = valid.Validate(cfg); err != nil {
		log.Error("配置校验失败", "error", err)
		return err
	}
	log.Info("配置文件加载成功", "config", cfg)

	agentHandler := ship.Default()
	exposeHandler := ship.Default()
	managerHandler := ship.Default()

	shipLog := shipx.NewLog(logHandler)
	agentHandler.Logger = shipLog
	agentHandler.Validator = valid
	agentHandler.NotFound = shipx.NotFound
	agentHandler.HandleError = shipx.HandleError
	exposeHandler.Logger = shipLog
	exposeHandler.Validator = valid
	exposeHandler.NotFound = shipx.NotFound
	exposeHandler.HandleError = shipx.HandleError
	managerHandler.Logger = shipLog
	managerHandler.Validator = valid
	managerHandler.NotFound = shipx.NotFound
	managerHandler.HandleError = shipx.HandleError

	secret := string(cfg.Secret)
	clientdCfg := clientd.Config{
		Secret:    secret,
		Semver:    cfg.Semver,
		Addresses: cfg.Addresses,
	}
	clientdOpt := clientd.NewOption().
		Logger(log).Handler(managerHandler)
	mux, dbc, err := clientd.Open(ctx, clientdCfg, clientdOpt)
	if err != nil {
		log.Error("broker 通道建立失败", "error", err)
		return err
	}
	if err = valid.Validate(dbc); err != nil {
		log.Error("管理端返回的基础报文不合法", "error", err)
	}

	gormLogCfg := gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		LogLevel:                  gormlogger.Info,
	}
	gormLog := logger.NewGorm(logHandler, gormLogCfg)
	db, err := sqldb.Open(dbc.DSN, &gorm.Config{Logger: gormLog})
	if err != nil {
		log.Error("连接数据发生错误", "error", err)
		return err
	}

	if sdb, _ := db.DB(); sdb != nil {
		defer sdb.Close() // 结束时释放数据库连接。
		sdb.SetMaxIdleConns(dbc.MaxIdleConn)
		sdb.SetMaxOpenConns(dbc.MaxOpenConn)
		sdb.SetConnMaxLifetime(dbc.MaxLifeTime)
		sdb.SetConnMaxIdleTime(dbc.MaxIdleTime)
	}
	log.Info("数据库连接成功", "dialect", db.Dialector.Name())

	pool := gopool.New(4096)
	qry := query.Use(db)
	tbl := qry.Broker
	dao := tbl.WithContext(ctx)
	thisBroker, err := dao.Where(tbl.Secret.Eq(secret)).First()
	if err != nil {
		log.Error("查询 broker 自身信息错误", "error", err)
		return err
	}

	// FIXME 开发时使用 safeMap，调试查看 map 内部变量比较方便。
	huber := linkhub.NewSafeMap(16)
	systemDialer := new(net.Dialer)
	agentDialer := linkhub.NewSuffixDialer(linkhub.AgentHostSuffix, huber)
	managerDialer := clientd.NewEqualDialer(mux, linkhub.ServerHost)
	multiDialer := linkhub.NewMatchedDialer(systemDialer, managerDialer, agentDialer)
	multiHTTP := &http.Client{Transport: &http.Transport{
		DialContext:           multiDialer.DialContext,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       time.Minute,
		ResponseHeaderTimeout: time.Minute,
	}}
	httpClient := httpkit.NewClient(multiHTTP)

	agentRPC := agtrpc.NewClient(httpClient)
	agentNotifierSvc := agentsvc.NewAgentNotifier(qry, agentRPC, pool, log)

	serverdOpt := serverd.NewOption().
		Logger(log).
		Handler(agentHandler).
		Valid(valid.Validate).
		Huber(huber).
		AgentNotifier(agentNotifierSvc)
	agentTunnelServer := serverd.New(qry, thisBroker, serverdOpt)

	const consoleDir = "resources/agent/console"
	if err = os.MkdirAll(consoleDir, 0o777); err != nil {
		return err
	}
	pipeFS := pipelog.NewFS(consoleDir, 1024*1024, time.Minute)
	defer pipeFS.Close()

	{
		routes := []shipx.RouteBinder{
			agentapi.NewAgentConsole(pipeFS),
			agentapi.NewPing(),
		}
		baseAPI := agentHandler.Group("/api/v1")
		if err = shipx.BindRoutes(baseAPI, routes); err != nil {
			log.Error("初始化 agent 路由出错", "error", err)
			return err
		}
	}
	{
		minionSvc := exposesvc.NewMinion(qry, log)
		_ = minionSvc.Reset(ctx, thisBroker.ID)
		defer minionSvc.Reset(ctx, thisBroker.ID) // FIXME ctx 可能已经取消了

		tunnelUpgrade := linkhub.NewHTTP(agentTunnelServer)
		routes := []shipx.RouteBinder{
			exposeapi.NewTunnel(tunnelUpgrade),
		}
		baseAPI := exposeHandler.Group("/api/v1")
		if err = shipx.BindRoutes(baseAPI, routes); err != nil {
			log.Error("初始化 expose 路由出错", "error", err)
			return err
		}
	}
	{

		routes := []shipx.RouteBinder{
			shipx.NewPprof(),
			managerapi.NewAgentConsole(pipeFS),
			managerapi.NewPing(),
		}
		baseAPI := managerHandler.Group("/api/v1")
		if err = shipx.BindRoutes(baseAPI, routes); err != nil {
			log.Error("初始化 manager 路由出错", "error", err)
			return err
		}
	}

	srv := &http.Server{
		Addr:    thisBroker.Bind,
		Handler: exposeHandler,
	}
	errs := make(chan error, 1)
	go serveHTTP(errs, srv)
	select {
	case err = <-errs:
	case <-ctx.Done():
		err = ctx.Err()
	}
	_ = srv.Close()

	return nil
}

func serveHTTP(errs chan<- error, srv *http.Server) {
	errs <- srv.ListenAndServe()
}
