package launch

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	agtrestapi "github.com/vela-ssoc/ssoc-broker/application/agent/restapi"
	agtservice "github.com/vela-ssoc/ssoc-broker/application/agent/service"
	"github.com/vela-ssoc/ssoc-broker/application/current/cronjob"
	curservice "github.com/vela-ssoc/ssoc-broker/application/current/service"
	"github.com/vela-ssoc/ssoc-broker/application/current/vmwrite"
	exprestapi "github.com/vela-ssoc/ssoc-broker/application/expose/restapi"
	mgtrestapi "github.com/vela-ssoc/ssoc-broker/application/manager/restapi"
	mgtservice "github.com/vela-ssoc/ssoc-broker/application/manager/service"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/agtaccept"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokcli"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/mgtclient"
	"github.com/vela-ssoc/ssoc-common/appcfg"
	"github.com/vela-ssoc/ssoc-common/banner"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/mongodb"
	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/vela-ssoc/ssoc-common/preadtls"
	"github.com/vela-ssoc/ssoc-common/shipx"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"github.com/vela-ssoc/ssoc-common/tlscert"
	"github.com/vela-ssoc/ssoc-common/validation"
	"github.com/vela-ssoc/ssoc-common/vmetric"
	"github.com/vela-ssoc/ssoc-proto/muxconn"
	"github.com/vela-ssoc/ssoc-proto/muxproto"
	"github.com/vela-ssoc/ssoc-proto/muxtool"
	"github.com/vela-ssoc/ssoc-proto/stegano"
	"github.com/xgfone/ship/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"gopkg.in/natefinch/lumberjack.v2"
)

func Exec(ctx context.Context, cfg string) error {
	var acr appcfg.Reader[config.Hide]
	if cfg != "" {
		acr = appcfg.NewJSON[config.Hide](cfg)
	} else {
		acr = stegano.Binary[config.Hide](os.Args[0])
	}

	return Run(ctx, acr)
}

//goland:noinspection GoUnhandledErrorResult
func Run(ctx context.Context, acr appcfg.Reader[config.Hide]) error {
	// 项目启动时还未连接到中心端，此时要默认一个日志输出。
	loglevel := new(slog.LevelVar) // 默认 INFO
	logoptions := &slog.HandlerOptions{AddSource: true, Level: loglevel}
	bootlogfile := &lumberjack.Logger{
		Filename:   "resources/log/application.jsonl",
		MaxSize:    100,
		MaxBackups: 10,
		LocalTime:  true,
		Compress:   true,
	}
	loghandlers := logger.NewMultiHandler(
		slog.NewJSONHandler(bootlogfile, logoptions), // 输出到文件
		slog.NewTextHandler(os.Stdout, logoptions),   // 输出到控制台
	)
	log := slog.New(loghandlers)
	log.Info("日志组件准备完毕（临时）")

	valid := validation.New()
	if err := valid.RegisterCustomValidations(validation.All()); err != nil {
		log.Error("校验器注册出错", "error", err)
		return err
	}

	hide, err := acr.Read(ctx)
	if err != nil {
		log.Error("读取隐写配置出错", "error", err)
		return err
	}
	if err = valid.Validate(hide); err != nil {
		log.Error("隐写配置校验出错", "error", err)
		return err
	}
	log.Info("隐写配置读取成功")

	shipLog := logger.NewFormat(loghandlers, 6)
	shipErr := shipx.NewErrorHandler(log)
	mgtSH := ship.Default()
	mgtSH.Validator = valid
	mgtSH.Logger = shipLog
	mgtSH.NotFound = shipErr.NotFound
	mgtSH.HandleError = shipErr.HandleError

	httpSH := ship.Default()
	httpSH.Validator = valid
	httpSH.Logger = shipLog
	httpSH.NotFound = shipErr.NotFound
	httpSH.HandleError = shipErr.HandleError

	httpsSH := ship.Default()
	httpsSH.Validator = valid
	httpsSH.Logger = shipLog
	httpsSH.NotFound = shipErr.NotFound
	httpsSH.HandleError = shipErr.HandleError

	agtSH := ship.Default()
	agtSH.Validator = valid
	agtSH.Logger = shipLog
	agtSH.NotFound = shipErr.NotFound
	agtSH.HandleError = shipErr.HandleError

	semver := banner.Version()
	brokOpts := brokcli.Options{
		Secret:     hide.Secret,
		Addresses:  hide.Addresses,
		Semver:     semver,
		Handler:    mgtSH,
		Validator:  valid.Validate,
		DialConfig: muxconn.DialConfig{Logger: log},
	}
	mux, err := brokcli.Open(ctx, brokOpts)
	if err != nil {
		log.Error("连接中心端失败", "error", err)
		return err
	}
	log.Info("连接中心端成功")

	bootCfg := mux.Config()
	log.Info("开始连接接数据库")
	mdb, err := mongodb.Connect(bootCfg.URI)
	if err != nil {
		log.Error("数据库连接错误", "error", err)
		return err
	}
	db := repository.NewDB(mdb, log)
	log.Info("数据库连接成功")

	this, err := db.Broker().FindBySecret(ctx, hide.Secret)
	if err != nil {
		log.Error("获取当前 broker 信息错误", "error", err)
		return err
	}

	thisID, cfg := this.ID, this.Config
	{
		// 初始化 logger
		lcfg := cfg.Logger
		_ = loglevel.UnmarshalText([]byte(lcfg.Level))
		loghandlers.Replace()
		if lcfg.Console {
			out := logger.NewTint(os.Stdout, logoptions)
			loghandlers.Append(out)
		}
		if name := lcfg.Filename; name != "" {
			lumber := &lumberjack.Logger{
				Filename:   name,
				MaxSize:    lcfg.MaxSize,
				MaxAge:     lcfg.MaxAge,
				MaxBackups: lcfg.MaxBackups,
				LocalTime:  lcfg.LocalTime,
				Compress:   lcfg.Compress,
			}
			defer lumber.Close()

			out := slog.NewJSONHandler(lumber, logoptions)
			loghandlers.Append(out)
		}
		bootlogfile.Close() // 关闭临时输出
	}
	log.Info("日志初始化完毕")

	hub := muxserver.NewAgentHub()
	muxopen := muxproto.NewMUXOpener(mux, muxproto.ManagerDomain)
	sysdial := new(net.Dialer)
	mixdial := muxserver.NewMixedDialer(muxopen, hub, sysdial)
	basecli := muxtool.NewClient(mixdial, log)
	mgtcli := mgtclient.NewClient(basecli)

	curBrokerSvc := curservice.NewBroker(db, thisID, log)
	curPyroscopeConfigSvc := curservice.NewPyroscopeConfig(db, thisID, log)
	curLokiConfigSvc := curservice.NewLokiConfig(db, thisID, logoptions, loghandlers, log)

	if err1 := curPyroscopeConfigSvc.Start(ctx); err1 != nil {
		log.Warn("启动 pyroscope 出错", "error", err1)
	}
	if err1 := curLokiConfigSvc.Start(ctx); err1 != nil {
		log.Warn("启动 loki 出错", "error", err1)
	}

	metricLabel := vmetric.BrokerLabel(thisID.Hex(), this.Name)
	curVictoriaMetricsSvc := curservice.NewVictoriaMetricsConfig(db, metricLabel, log)
	mgtTunnelSvc := mgtservice.NewTunnel(mux, log)

	acptOpt := agtaccept.Options{
		Huber:      hub,
		Handler:    agtSH,
		Validator:  valid.Validate,
		Logger:     log,
		BootLoader: nil,
		ThisBroker: func() (bson.ObjectID, string) {
			return thisID, this.Name
		},
		Notifier: nil,
	}
	agtAcpt := agtaccept.NewAccept(db, acptOpt)
	tunnelV1API := exprestapi.NewTunnelV1(agtAcpt)

	// httpRoutes 和 httpsRoutes 均为需要暴露的路由。
	// 由于 http 不安全，所以仅挂载必要的 agent 兼容业务。
	httpRoutes := []shipx.RouteRegister{
		exprestapi.NewHeartbeat(),
		tunnelV1API,
	}
	httpsRoutes := []shipx.RouteRegister{
		exprestapi.NewTunnel(agtAcpt), // 新版 tunnel 仅支持 https
		tunnelV1API,
	}
	mgtRoutes := []shipx.RouteRegister{
		mgtrestapi.NewSpeedtest(),
		mgtrestapi.NewTunnel(mgtTunnelSvc),
	}

	agtHeartbeatV1Svc := agtservice.NewHeartbeatV1(db, log)
	agtSysInfoV1Svc := agtservice.NewSysInfoV1(db, log)
	agtRoutes := []shipx.RouteRegister{
		agtrestapi.NewHeartbeatV1(agtHeartbeatV1Svc),
		agtrestapi.NewSysInfoV1(agtSysInfoV1Svc),
	}
	{
		base := httpSH.Group("/api/v1")
		if err = shipx.RegisterRoutes(base, httpRoutes); err != nil {
			log.Error("注册 http 路由出错", "error", err)
			return err
		}
	}
	{
		routes := append(httpsRoutes, httpRoutes...)
		base := httpsSH.Group("/api/v1")
		if err = shipx.RegisterRoutes(base, routes); err != nil {
			log.Error("注册 https 路由出错", "error", err)
			return err
		}
	}
	{
		base := mgtSH.Group("/api/v1")
		if err = shipx.RegisterRoutes(base, mgtRoutes); err != nil {
			log.Error("注册 manager 路由出错", "error", err)
			return err
		}
	}
	{
		base := agtSH.Group("/api/v1")
		if err = shipx.RegisterRoutes(base, agtRoutes); err != nil {
			log.Error("注册 agent 路由出错", "error", err)
			return err
		}
	}

	metricWriters := []vmetric.MetricWriter{
		vmetric.NewPsutil(),
		vmwrite.NewTunnel(mux),
	}

	cronTasks := []cronv3.Tasker{
		cronjob.NewAgentTunnelStat(db, hub, basecli),
		cronjob.NewHeartbeat(mgtcli, log),
		cronjob.NewMetrics(curVictoriaMetricsSvc, metricWriters),
		cronjob.NewTunnelStat(db, thisID, mux),
	}

	crontab := cronv3.New(log)
	crontab.Start()
	if err = crontab.AddTasks(cronTasks); err != nil {
		log.Error("定时任务注册出错", "error", err)
		return err
	}

	addr := cfg.Server.Addr
	if addr == "" {
		addr = ":443"
	}
	lis, err := preadtls.ListenTCP(addr, 10*time.Second)
	if err != nil {
		log.Error("服务监听出错", "error", err)
		return err
	}
	defer lis.Close()

	if err = curBrokerSvc.ResetAgents(time.Minute); err != nil {
		log.Error("重置节点状态出错", "error", err)
		return err
	}

	crtPool := tlscert.NewMatch(db.Certificate(), log)
	httpSrv := &http.Server{Handler: httpSH}
	httpsSrv := &http.Server{Handler: httpsSH, TLSConfig: &tls.Config{GetCertificate: crtPool.GetCertificate}}
	errs := make(chan error, 1)
	go serveHTTP(errs, httpSrv, lis.TCPListener())
	go serveHTTPS(errs, httpsSrv, lis.TLSListener())

	select {
	case <-ctx.Done():
	case err = <-errs:
	}

	crontab.Stop()
	_ = httpSrv.Close()
	_ = httpsSrv.Close()
	_ = curBrokerSvc.ResetAgents(10 * time.Second)
	_ = mux.Close()
	_ = curPyroscopeConfigSvc.Close()
	_ = curLokiConfigSvc.Close()

	cause := context.Cause(ctx)
	log.Error("程序停止运行", "error", err, "cause", cause)

	return err
}

func serveHTTP(errs chan<- error, srv *http.Server, ln net.Listener) {
	errs <- srv.Serve(ln)
}

func serveHTTPS(errs chan<- error, srv *http.Server, ln net.Listener) {
	errs <- srv.ServeTLS(ln, "", "")
}
