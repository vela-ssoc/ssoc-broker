package launch

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	curservice "github.com/vela-ssoc/ssoc-broker/application/current/service"
	exprestapi "github.com/vela-ssoc/ssoc-broker/application/expose/restapi"
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokcli"
	"github.com/vela-ssoc/ssoc-common/appcfg"
	"github.com/vela-ssoc/ssoc-common/datalayer/query"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/preadtls"
	"github.com/vela-ssoc/ssoc-common/shipx"
	"github.com/vela-ssoc/ssoc-common/sqldb"
	"github.com/vela-ssoc/ssoc-common/tlscert"
	"github.com/vela-ssoc/ssoc-common/validation"
	"github.com/vela-ssoc/ssoc-proto/muxconn"
	"github.com/vela-ssoc/ssoc-proto/stegano"
	"github.com/xgfone/ship/v5"
	"gopkg.in/natefinch/lumberjack.v2"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

func Run(ctx context.Context, cfg string) error {
	var acr appcfg.Reader[config.Hide]
	if cfg != "" {
		acr = appcfg.NewJSON[config.Hide](cfg)
	} else {
		acr = stegano.Binary[config.Hide](os.Args[0])
	}

	return Exec(ctx, acr)
}

//goland:noinspection GoUnhandledErrorResult
func Exec(ctx context.Context, acr appcfg.Reader[config.Hide]) error {
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

	shipLog := logger.NewShip(logh)
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

	brokOpts := brokcli.Options{
		Secret:    hide.Secret,
		Addresses: hide.Addresses,
		Semver:    hide.Semver,
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
	defer mux.Close()
	log.Info("连接中心端成功")

	bootCfg := mux.Config()
	gormLogCfg := gormlogger.Config{LogLevel: gormlogger.Info}
	gormLog := logger.NewGorm(logh, gormLogCfg)
	gormCfg := &gorm.Config{Logger: gormLog}
	db, err := sqldb.Open(bootCfg.DSN, gormCfg)
	if err != nil {
		log.Error("连接数据库失败", "error", err)
		return err
	}
	qry := query.Use(db)

	curBrokerSvc := curservice.NewBroker(hide.Secret, qry, log)

	// httpRoutes 和 httpsRoutes 均为需要暴露的路由。
	// 由于 http 不安全，所以仅挂载必要的 agent 兼容业务。
	httpRoutes := []shipx.RouteBinder{
		exprestapi.NewHealth(),
	}
	httpsRoutes := []shipx.RouteBinder{}
	mgtRoutes := []shipx.RouteBinder{}
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

	lis, err := preadtls.ListenTCP(":8082", 10*time.Second)
	if err != nil {
		log.Error("服务监听出错", "error", err)
		return err
	}
	defer lis.Close()

	if err = curBrokerSvc.ResetAgents(time.Minute); err != nil {
		log.Error("重置节点状态出错", "error", err)
		return err
	}

	crtPool := tlscert.NewMatch(noneTLS, log)
	httpSrv := &http.Server{Handler: httpSH}
	httpsSrv := &http.Server{Handler: httpsSH, TLSConfig: &tls.Config{GetCertificate: crtPool.GetCertificate}}
	errs := make(chan error, 1)
	go serveHTTP(errs, httpSrv, lis.TCPListener())
	go serveHTTPS(errs, httpsSrv, lis.TLSListener())

	select {
	case <-ctx.Done():
	case err = <-errs:
	}

	_ = httpSrv.Close()
	_ = httpsSrv.Close()
	_ = curBrokerSvc.ResetAgents(10 * time.Second)

	return err
}

func serveHTTP(errs chan<- error, srv *http.Server, ln net.Listener) {
	errs <- srv.Serve(ln)
}

func serveHTTPS(errs chan<- error, srv *http.Server, ln net.Listener) {
	errs <- srv.ServeTLS(ln, "", "")
}

func noneTLS(context.Context) ([]*tls.Certificate, error) {
	return nil, nil
}
