package banner

import (
	"fmt"
	"io"
	"os"
	"os/user"
	"runtime"
	"runtime/debug"
	"sync/atomic"
	"time"
)

const logo = "\u001B[1;33m" +
	"   ______________  _____  \n" +
	"  / ___/ ___/ __ \\/ ___/ \n" +
	" (__  |__  ) /_/ / /__    \n" +
	"/____/____/\\____/\\___/  \u001B[0m  \u001B[1;32mBROKER\u001B[0m\n" +
	"Powered By: 东方财富安全团队\n\n" +
	"    操作系统: \u001B[1;1m%s\u001B[0m\n" +
	"    系统架构: \u001B[1;1m%s\u001B[0m\n" +
	"    主机名称: \u001B[1;1m%s\u001B[0m\n" +
	"    当前用户: \u001B[1;1m%s\u001B[0m\n" +
	"    进程 PID: \u001B[1;1m%d\u001B[0m\n" +
	"    软件版本: \u001B[1;1m%s\u001B[0m\n" +
	"    编译时间: \u001B[1;1m%s\u001B[0m\n" +
	"    提交时间: \u001B[1;1m%s\u001B[0m\n" +
	"    修订版本: \u001B[1;1m%s\u001B[0m\n\n\n"

var (
	// version 项目发布版本号
	// 项目每次发布版本后会打一个 tag, 这个版本号就来自 git 最新的 tag
	version string

	// revision 修订版本, 代码最近一次的提交 ID
	revision string

	// compileAt 编译时间, 由编译脚本在编译时注入
	// 无论 macOS 还是 Linux 下, date 命令的默认格式都遵循 time.UnixDate
	compileTime string

	compileAt time.Time

	// commitAt 代码最近一次提交时间
	commitAt time.Time

	// pid 进程 ID
	pid int

	// username 当前系统用户名
	username string

	// hostname 主机名
	hostname string

	parsed atomic.Bool
)

// WriteTo 打印 banner 到指定输出流
func WriteTo(w io.Writer) {
	parse()

	compile := compileTime
	if !compileAt.IsZero() {
		compile = compileAt.In(time.Local).String()
	}

	var commit string
	if !commitAt.IsZero() {
		commit = commitAt.In(time.Local).String()
	}

	_, _ = fmt.Fprintf(w, logo, runtime.GOOS, runtime.GOARCH, hostname,
		username, pid, version, compile, commit, revision)
}

func parse() {
	if !parsed.CompareAndSwap(false, true) {
		return
	}

	pid = os.Getpid() // 获取 PID
	if cu, _ := user.Current(); cu != nil {
		username = cu.Username
	}
	hostname, _ = os.Hostname()

	// time.RFC1123Z Linux `date -R` 输出的时间格式
	// time.UnixDate macOS `date` 输出的时间格式
	layouts := []string{time.RFC1123Z, time.UnixDate}
	for _, layout := range layouts {
		if at, err := time.Parse(layout, compileTime); err == nil {
			compileAt = at
			break
		}
	}

	info, _ := debug.ReadBuildInfo()
	if info == nil {
		return
	}

	if version == "" {
		version = info.Main.Version
	}

	settings := info.Settings
	for _, set := range settings {
		key, val := set.Key, set.Value
		switch key {
		case "vcs.revision":
			revision = val
		case "vcs.time":
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				commitAt = t
			}
		}
	}
}
