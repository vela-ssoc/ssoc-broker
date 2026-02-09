package launch

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/vela-ssoc/ssoc-broker/application/current/cronjob"
	curservice "github.com/vela-ssoc/ssoc-broker/application/current/service"
	"github.com/vela-ssoc/ssoc-broker/application/current/vmwrite"
	exprestapi "github.com/vela-ssoc/ssoc-broker/application/expose/restapi"
	mgtrestapi "github.com/vela-ssoc/ssoc-broker/application/manager/restapi"
	mgtservice "github.com/vela-ssoc/ssoc-broker/application/manager/service"
	"github.com/vela-ssoc/ssoc-broker/config"
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
	logOpts := &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
	tmpLumber := &lumberjack.Logger{
		Filename:   "resources/log/application.jsonl",
		MaxSize:    100,
		MaxBackups: 10,
		LocalTime:  true,
		Compress:   true,
	}
	logh := logger.NewMultiHandler(
		logger.NewTint(os.Stdout, logOpts),
		slog.NewJSONHandler(tmpLumber, logOpts),
	)
	log := slog.New(logh)
	log.Info("初始日志组件装配完毕")

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

	shipLog := logger.NewFormat(logh, 6)
	mgtSH := ship.Default()
	mgtSH.Validator = valid
	mgtSH.Logger = shipLog

	httpSH := ship.Default()
	httpSH.Validator = valid
	httpSH.Logger = shipLog

	httpsSH := ship.Default()
	httpsSH.Validator = valid
	httpsSH.Logger = shipLog

	agtSH := ship.Default()
	agtSH.Validator = valid
	agtSH.Logger = shipLog

	semver := banner.Version()
	brokOpts := brokcli.Options{
		Secret:    hide.Secret,
		Addresses: hide.Addresses,
		Semver:    semver,
		Handler:   mgtSH,
		Validator: valid.Validate,
		DialConfig: muxconn.DialConfig{
			Protocol: hide.Protocol,
			Logger:   log,
		},
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

	curBrokerSvc := curservice.NewBroker(db, hide.Secret, log)
	this, err := curBrokerSvc.Get(ctx)
	if err != nil {
		log.Error("获取当前 broker 信息错误", "error", err)
		return err
	}

	cfg := this.Config
	{
		// 初始化 logger
		lcfg := cfg.Logger
		level := new(slog.LevelVar)
		if e := level.UnmarshalText([]byte(lcfg.Level)); e != nil {
			level.Set(slog.LevelInfo)
		}
		opts := &slog.HandlerOptions{AddSource: true, Level: level}
		logh.Replace()
		if lcfg.Console {
			out := logger.NewTint(os.Stdout, opts)
			logh.Append(out)
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

			out := slog.NewJSONHandler(lumber, opts)
			logh.Append(out)
		}
		tmpLumber.Close() // 关闭临时输出
	}
	log.Info("日志初始化完毕")

	muxopen := muxproto.NewMUXOpener(mux, muxproto.ManagerDomain)
	sysdial := new(net.Dialer)
	mixdial := muxserver.NewMixedDialer(muxopen, nil, sysdial)
	basecli := muxtool.NewClient(mixdial, log)
	mgtcli := mgtclient.NewClient(basecli)

	curPyroscopeSvc := curservice.NewPyroscopeConfig(db, this.ID, log)
	if err1 := curPyroscopeSvc.Start(ctx); err1 != nil {
		log.Warn("启动 pyroscope 出错", "error", err1)
	}

	curVictoriaMetricsSvc := curservice.NewVictoriaMetricsConfig(db, this, log)
	mgtTunnelSvc := mgtservice.NewTunnel(mux, log)

	// httpRoutes 和 httpsRoutes 均为需要暴露的路由。
	// 由于 http 不安全，所以仅挂载必要的 agent 兼容业务。
	httpRoutes := []shipx.RouteBinder{
		exprestapi.NewHeartbeat(),
	}
	httpsRoutes := []shipx.RouteBinder{}
	mgtRoutes := []shipx.RouteBinder{
		mgtrestapi.NewSpeedtest(),
		mgtrestapi.NewTunnel(mgtTunnelSvc),
	}
	agtRoutes := []shipx.RouteBinder{}
	{
		base := httpSH.Group("/api/v1")
		if err = shipx.BindRoutes(base, httpRoutes); err != nil {
			log.Error("注册 http 路由出错", "error", err)
			return err
		}
	}
	{
		routes := append(httpsRoutes, httpRoutes...)
		base := httpsSH.Group("/api/v1")
		if err = shipx.BindRoutes(base, routes); err != nil {
			log.Error("注册 https 路由出错", "error", err)
			return err
		}
	}
	{
		base := mgtSH.Group("/api/v1")
		if err = shipx.BindRoutes(base, mgtRoutes); err != nil {
			log.Error("注册 manager 路由出错", "error", err)
			return err
		}
	}
	{
		base := mgtSH.Group("/api/v1")
		if err = shipx.BindRoutes(base, agtRoutes); err != nil {
			log.Error("注册 agent 路由出错", "error", err)
			return err
		}
	}

	metricWriters := []vmetric.MetricWriter{
		vmetric.NewPsutil(),
		vmwrite.NewTunnel(mux),
	}

	cronTasks := []cronv3.Tasker{
		cronjob.NewHeartbeat(mgtcli, log),
		cronjob.NewMetrics(curVictoriaMetricsSvc, metricWriters),
		cronjob.NewTunnelStat(db, this.ID, mux),
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

	crtPool := tlscert.NewMatch(noneTLS{}, log)
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
	_ = curPyroscopeSvc.Stop()

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

type noneTLS struct{}

func (noneTLS) LoadCertificate(context.Context) ([]*tls.Certificate, error) {
	return nil, nil
}
