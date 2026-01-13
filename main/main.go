package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"

	"github.com/vela-ssoc/ssoc-broker/hideconf"
	"github.com/vela-ssoc/ssoc-broker/launch"
	"github.com/vela-ssoc/ssoc-common/banner"
)

func main() {
	var version bool
	var config string
	flag.BoolVar(&version, "v", false, "打印版本号")
	if hideconf.DevMode { // 开发模式：go build -tags=dev
		flag.StringVar(&config, "c", "broker.jsonc", "配置文件")
	}
	flag.Parse()

	if _, _ = banner.ANSI(os.Stdout); version {
		return
	}

	hide, err := hideconf.Read(config)
	if err != nil {
		slog.Error("读取 hide 配置错误", slog.Any("error", err), slog.String("config", config))
		return
	}

	slog.Info("按 Ctrl+C 结束运行")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err = launch.Run(ctx, hide); err != nil {
		slog.Error("程序运行错误", slog.Any("error", err))
	}

	slog.Info("程序运行结束")
}
