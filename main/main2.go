package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"syscall"

	"github.com/vela-ssoc/ssoc-broker/launch2"
	"github.com/vela-ssoc/ssoc-common/banner"
)

func main1() {
	args := os.Args
	name := filepath.Base(args[0])
	set := flag.NewFlagSet(name, flag.ExitOnError)
	ver := set.Bool("v", false, "打印版本")
	cfg := set.String("c", "resources/config/", "配置文件")
	_ = set.Parse(args[1:])
	if _, _ = banner.ANSI(os.Stdout); *ver {
		return
	}

	for _, str := range []string{"resources/.crash.txt", ".crash.txt"} {
		if f, _ := os.Create(str); f != nil {
			_ = debug.SetCrashOutput(f, debug.CrashOptions{})
			_ = f.Close()
			break
		}
	}

	signals := []os.Signal{syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT}
	ctx, cancel := signal.NotifyContext(context.Background(), signals...)
	defer cancel()

	if err := launch2.Exec(ctx, *cfg); err != nil {
		slog.Error("服务运行错误", slog.Any("error", err))
	} else {
		slog.Info("服务停止运行")
	}
}
