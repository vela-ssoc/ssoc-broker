package service

import (
	"time"

	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
)

type PhaseService interface {
	mlink.NodePhaser
}

func NodeEvent(compare subtask.Comparer, pool taskpool.Executor, slog logback.Logger) PhaseService {
	return &nodeEventService{
		compare: compare,
		pool:    pool,
		slog:    slog,
	}
}

type nodeEventService struct {
	compare subtask.Comparer
	pool    taskpool.Executor
	slog    logback.Logger
}

func (biz *nodeEventService) Created(id int64, inet string, at time.Time) {
}

func (biz *nodeEventService) Repeated(id int64, ident gateway.Ident, at time.Time) {
}

func (biz *nodeEventService) Connected(lnk mlink.Linker, ident gateway.Ident, issue gateway.Issue, at time.Time) {
	mid, inet := issue.ID, ident.Inet.String()
	biz.slog.Infof("agent %s(%d) 上线了", inet, mid)
	task := subtask.SyncTask(lnk, biz.compare, mid, inet, biz.slog)
	biz.pool.Submit(task)
}

func (biz *nodeEventService) Disconnected(lnk mlink.Linker, ident gateway.Ident, issue gateway.Issue, at time.Time, du time.Duration) {
	mid, inet := issue.ID, ident.Inet.String()
	biz.slog.Warnf("agent %s(%d) 下线了", inet, mid)
}
