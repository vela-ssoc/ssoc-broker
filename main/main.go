package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-broker/launch"
	"github.com/vela-ssoc/ssoc-common/banner"
)

func main() {
	args := os.Args
	flg := flag.NewFlagSet(args[0], flag.ExitOnError)
	ver := flg.Bool("v", false, "打印版本号并退出")
	cfg := flg.String("c", "resources/config/application.jsonc", "配置文件")
	_ = flg.Parse(args[1:])

	if _, _ = banner.ANSI(os.Stdout); *ver {
		return
	}

	fmt.Println("按 Ctrl+C 结束运行")
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// 如果当前目录存在配置文件，就先读取本地文件。不存在则读取二进制内的隐写数据。
	cfgs := config.Parsers{config.JSONC(*cfg), config.Binary(args[0])}
	if err := launch.Run(ctx, cfgs); err != nil {
		fmt.Printf("程序运行出错 err=%q", err)
	}
	fmt.Println("程序结束运行")
}
