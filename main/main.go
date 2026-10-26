package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"github.com/vela-ssoc/ssoc-broker/launch"
	"github.com/vela-ssoc/ssoc-common-mb/bininfo"
)

func main() {
	args := os.Args
	flg := flag.NewFlagSet(args[0], flag.ExitOnError)
	ver := flg.Bool("v", false, "打印版本号并退出")
	cfg := flg.String("c", "resources/config/application.jsonc", "配置文件")
	_ = flg.Parse(args[1:])

	if _, _ = bininfo.ANSI(os.Stdout); *ver {
		return
	}

	fmt.Println("按 Ctrl+C 结束运行")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	slog.Info("按 Ctrl+C 停止运行")
	err := launch.RunV3(ctx, *cfg)
	cause := context.Cause(ctx)
	slog.Warn("程序停止运行", "error", err, "cause", cause)
}
