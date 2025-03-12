package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/vela-ssoc/vela-broker/banner"
	"github.com/vela-ssoc/vela-broker/hideconf"
	"github.com/vela-ssoc/vela-broker/launch"
)

func main() {
	var version bool
	var config string
	flag.BoolVar(&version, "v", false, "打印版本号")
	if hideconf.DevMode { // 开发模式：go build -tags=dev
		flag.StringVar(&config, "c", "broker.jsonc", "配置文件")
	}
	flag.Parse()

	if banner.ANSI(os.Stdout); version {
		return
	}

	logOpt := &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug}
	jsonHandler := slog.NewJSONHandler(os.Stdout, logOpt)
	log := slog.New(jsonHandler)

	hide, err := hideconf.Read(config)
	if err != nil {
		log.Error("读取 hide 配置错误", slog.Any("error", err), slog.String("config", config))
		return
	}

	cares := []os.Signal{syscall.SIGTERM, syscall.SIGHUP, syscall.SIGKILL, syscall.SIGINT}
	ctx, cancel := signal.NotifyContext(context.Background(), cares...)
	defer cancel()
	log.Info("按 Ctrl+C 结束运行")

	if err = launch.Run(ctx, hide); err != nil {
		log.Error("程序运行错误", slog.Any("error", err))
	}

	log.Info("程序运行结束")
}
