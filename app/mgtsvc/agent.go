package mgtsvc

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/storage/v2"
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

func Agent(qry *query.Query, lnk mlink.Linker, mon MinionService, store storage.Storer, log *slog.Logger) AgentService {
	return &agentService{
		qry:   qry,
		lnk:   lnk,
		mon:   mon,
		store: store,
		log:   log,
		pool:  gopool.NewV2(512),
		cycle: 5,
	}
}

type agentService struct {
	qry   *query.Query
	lnk   mlink.Linker
	mon   MinionService
	store storage.Storer
	log   *slog.Logger
	pool  gopool.Pool
	cycle int
}
