package launch

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokcli"
	"github.com/vela-ssoc/ssoc-common/appcfg"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/sqldb"
	"github.com/vela-ssoc/ssoc-common/validation"
	"github.com/vela-ssoc/ssoc-proto/muxconn"
	"github.com/vela-ssoc/ssoc-proto/stegano"
	"github.com/xgfone/ship/v5"
	"gopkg.in/natefinch/lumberjack.v2"
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

	mgtSH := ship.Default()
	httpsSH := ship.Default()
	mgtSH.Validator = valid
	httpsSH.Validator = valid

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
	//goland:noinspection GoUnhandledErrorResult
	defer mux.Close()
	log.Info("连接中心端成功")

	bootCfg := mux.Config()
	db, err := sqldb.Open(bootCfg.DSN)
	if err != nil {
		log.Error("连接数据库失败", "error", err)
		return err
	}
	_ = db

	return http.ListenAndServe(":8082", httpsSH)
}
