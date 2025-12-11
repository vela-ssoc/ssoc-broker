package execute

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/vela-ssoc/ssoc-broker/application/current"
	expresetapi "github.com/vela-ssoc/ssoc-broker/application/expose/restapi"
	"github.com/vela-ssoc/ssoc-broker/channel/clientd"
	"github.com/vela-ssoc/ssoc-broker/channel/serverd"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common/httpkit"
	"github.com/vela-ssoc/ssoc-common/linkhub"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/preadtls"
	"github.com/vela-ssoc/ssoc-common/profile"
	"github.com/vela-ssoc/ssoc-common/shipx"
	"github.com/vela-ssoc/ssoc-common/sqldb"
	"github.com/vela-ssoc/ssoc-common/stegano"
	"github.com/vela-ssoc/ssoc-common/tlscert"
	"github.com/vela-ssoc/ssoc-common/validation"
	"github.com/xgfone/ship/v5"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Exec(ctx context.Context, configFile string) error {
	var pld profile.Reader[config.Config]
	if configFile != "" {
		pld = profile.NewFile[config.Config](configFile)
	} else {
		pld = stegano.File[config.Config](os.Args[0])
	}

	return exec(ctx, pld)
}

func exec(ctx context.Context, pld profile.Reader[config.Config]) error {
	// 初始化启动日志
	logh := logger.Multi(logger.NewTint(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug, AddSource: true}))
	log := slog.New(logh)

	valid := validation.New()
	_ = valid.RegisterCustomValidations(validation.All())

	cfg, err := pld.Read(ctx)
	if err != nil {
		log.Error("配置读取错误", "error", err)
		return err
	}
	if err = valid.Validate(cfg); err != nil {
		log.Error("配置校验错误", "error", err, "config", cfg)
		return err
	}
	log.Info("配置文件加载并校验通过")

	agentHandler := ship.Default()
	exposeTLSHandler := ship.Default()
	exposeTCPHandler := ship.Default()
	managerHandler := ship.Default()
	{
		shipLog := shipx.NewLog(logh)
		agentHandler.Logger = shipLog
		agentHandler.Validator = valid
		agentHandler.NotFound = shipx.NotFound
		agentHandler.HandleError = shipx.HandleError
		exposeTLSHandler.Logger = shipLog
		exposeTLSHandler.Validator = valid
		exposeTLSHandler.NotFound = shipx.NotFound
		exposeTLSHandler.HandleError = shipx.HandleError
		exposeTCPHandler.Logger = shipLog
		exposeTCPHandler.Validator = valid
		exposeTCPHandler.NotFound = shipx.NotFound
		exposeTCPHandler.HandleError = shipx.HandleError
		managerHandler.Logger = shipLog
		managerHandler.Validator = valid
		managerHandler.NotFound = shipx.NotFound
		managerHandler.HandleError = shipx.HandleError
	}

	clientdCfg := clientd.Config{Secret: cfg.Secret, Semver: cfg.Semver, Addresses: cfg.Addresses}
	clientdOpt := clientd.Options{Handler: managerHandler, Logger: log, Timeout: 10 * time.Second}
	mux, err := clientd.Open(ctx, clientdCfg, clientdOpt)
	if err != nil {
		log.Error("通道建立失败", "error", err)
		return err
	}
	bootConfig := mux.BootConfig()
	if err = valid.Validate(bootConfig); err != nil {
		log.Error("中心端下发的配置无效", "error", err, "boot_config", bootConfig)
		return err
	}
	log.Info("中心端下发的配置文件加载并校验通过")

	// 开始连接数据库
	gormLogCfg := gormlogger.Config{
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true,
		LogLevel:                  gormlogger.Info,
	}
	gormLog := logger.NewGorm(logh, gormLogCfg)
	db, err := sqldb.Open(bootConfig.DSN, &gorm.Config{Logger: gormLog})
	if err != nil {
		log.Error("连接数据错误", "error", err)
		return err
	}
	if sdb, err1 := db.DB(); err1 != nil {
		log.Error("获取数据库底层连接错误", "error", err1)
		return err1
	} else {
		//goland:noinspection GoUnhandledErrorResult
		defer sdb.Close()
		sdb.SetConnMaxLifetime(bootConfig.MaxLifeTime)
		sdb.SetMaxIdleConns(bootConfig.MaxIdleConn)
		sdb.SetMaxOpenConns(bootConfig.MaxOpenConn)
		sdb.SetConnMaxLifetime(bootConfig.MaxLifeTime)
	}
	qry := query.Use(db)
	log.Info("数据库连接成功", "dialect", db.Dialector.Name())

	currentBrokerSvc := current.NewBroker(cfg.Secret, qry, log)
	this, err := currentBrokerSvc.Load(ctx)
	if err != nil {
		log.Info("从数据库中查询当前 broker 错误", "error", err, "secret", cfg.Secret)
		return err
	}
	log.Info("当前 broker 验证通过", "name", this.Name, "bind", this.Bind)

	// 使用数据库中的日志配置
	// logh.Replace()

	// TODO 业务程序
	// FIXME 开发时使用 safeMap，调试查看 map 内部变量比较方便。
	huber := linkhub.NewSafeMap(16)
	// huber := linkhub.NewShardMap(4096) // 线上正式环境
	systemDialer := new(net.Dialer)
	agentDialer := linkhub.NewSuffixDialer(linkhub.AgentHostSuffix, huber)
	managerDialer := clientd.NewEqualDialer(mux, linkhub.ServerHost)
	selectDialer := linkhub.NewSelectDialer(systemDialer, managerDialer, agentDialer)
	multiHTTP := &http.Client{Transport: &http.Transport{
		DialContext:           selectDialer.DialContext,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		MaxConnsPerHost:       20,
		IdleConnTimeout:       time.Minute,
		ResponseHeaderTimeout: time.Minute,
	}}
	_ = httpkit.NewClient(multiHTTP)

	{
		serverdOpt := serverd.NewOption().
			Logger(log).
			Handler(agentHandler).
			Valid(valid.Validate).
			Huber(huber)
		agentTunnelServer := serverd.New(qry, this, serverdOpt)

		routes := []shipx.RouteBinder{
			expresetapi.NewTunnel(agentTunnelServer),
		}
		baseAPI := exposeTLSHandler.Group("/api/v1")
		if err = shipx.BindRoutes(baseAPI, routes); err != nil {
			log.Error("路由注册错误（expose-tls）", "error", err)
			return err
		}
	}
	{
		routes := []shipx.RouteBinder{
			expresetapi.NewTunnelOld(),
		}
		baseAPI := exposeTCPHandler.Group("/api/v1")
		if err = shipx.BindRoutes(baseAPI, routes); err != nil {
			log.Error("路由注册错误（expose-tcp）", "error", err)
			return err
		}
	}

	// TODO 业务程序
	certPool := tlscert.NewCertPool(currentBrokerSvc.Certificates, log)
	tlsConfig := &tls.Config{GetCertificate: certPool.Match}
	v1Log := logger.NewV1(log, 8)
	exposeSrv := &http.Server{
		Handler:   exposeTLSHandler,
		TLSConfig: tlsConfig,
		ErrorLog:  v1Log,
	}
	exposeTCPSrv := &http.Server{
		Handler:  exposeTCPHandler,
		ErrorLog: v1Log,
	}
	lis, err := net.Listen("tcp", this.Bind)
	if err != nil {
		return err
	}
	if err = currentBrokerSvc.ResetAgents(ctx); err != nil {
		log.Warn("启动前重置 agent 节点状态错误", "error", err)
	}

	errs := make(chan error)
	muxListen := preadtls.NewListener(lis, 10*time.Second)
	go serveHTTP(errs, exposeTCPSrv, muxListen.TCPListener())
	go serveHTTPS(errs, exposeSrv, muxListen.TLSListener())

	select {
	case err = <-errs:
	case <-ctx.Done():
		err = ctx.Err()
	}
	log.Warn("程序运行结束", "error", err)
	_ = exposeTCPSrv.Close()
	_ = exposeSrv.Close()
	_ = muxListen.Close()
	if err1 := currentBrokerSvc.ResetAgents(nil); err1 != nil {
		log.Warn("关闭程序前重置 agent 节点状态错误", "error", err1)
	}

	return err
}

func serveHTTP(errs chan<- error, srv *http.Server, lis net.Listener) {
	errs <- srv.Serve(lis)
}

func serveHTTPS(errs chan<- error, srv *http.Server, lis net.Listener) {
	errs <- srv.ServeTLS(lis, "", "")
}
