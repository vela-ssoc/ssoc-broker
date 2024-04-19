package mgtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb-itai/accord"
	"github.com/vela-ssoc/vela-common-mb-itai/gopool"
	"github.com/vela-ssoc/vela-common-mb-itai/logback"
	"github.com/vela-ssoc/vela-common-mb-itai/storage/v2"
)

type AgentService interface {
	// TableTask 同步表任务。
	TableTask(ctx context.Context, tid int64) error

	// RsyncTask 同步 agent 节点的配置。
	RsyncTask(ctx context.Context, mids []int64) error

	// ReloadTask 重新加载指定节点的指定配置。
	ReloadTask(ctx context.Context, mid, sid int64) error

	// ReloadStartup 重新加载指定节点的 startup 配置。
	ReloadStartup(ctx context.Context, mid int64) error

	// ThirdDiff 通知 agent 节点三方文件发生了变更。
	ThirdDiff(ctx context.Context, name, event string) error

	// Command 向节点发送命令
	Command(ctx context.Context, mids []int64, cmd string) error

	// Upgrade 向节点发送升级命令
	Upgrade(ctx context.Context, req *accord.Upgrade) error
}

func Agent(lnk mlink.Linker, mon MinionService, store storage.Storer, slog logback.Logger) AgentService {
	return &agentService{
		lnk:   lnk,
		mon:   mon,
		store: store,
		slog:  slog,
		pool:  gopool.New(50, 10, time.Minute),
		cycle: 3,
	}
}

type agentService struct {
	lnk   mlink.Linker
	mon   MinionService
	store storage.Storer
	slog  logback.Logger
	pool  gopool.Executor
	cycle int
}
