package service

import (
	"context"

	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
)

type TaskService interface {
	Sync(ctx context.Context, mid int64, inet string) error
	Load(ctx context.Context, mid, sid int64, inet string) error
}

func Task(lnk mlink.Linker, compare subtask.Comparer, pool taskpool.Executor, slog logback.Logger) TaskService {
	return &taskService{
		lnk:     lnk,
		compare: compare,
		pool:    pool,
		slog:    slog,
	}
}

type taskService struct {
	lnk     mlink.Linker
	compare subtask.Comparer
	pool    taskpool.Executor
	slog    logback.Logger
}

func (biz *taskService) Sync(ctx context.Context, mid int64, inet string) error {
	task := subtask.SyncTask(biz.lnk, biz.compare, mid, inet, biz.slog)
	biz.pool.Submit(task)
	return nil
}

func (biz *taskService) Load(ctx context.Context, mid, sid int64, inet string) error {
	// 查询配置信息
	tbl := query.Substance
	sub, err := tbl.WithContext(ctx).Where(tbl.ID.Eq(sid)).First()
	if err != nil {
		return err
	}

	task := subtask.DiffTask(biz.lnk, biz.compare, sub, mid, inet, biz.slog)
	biz.pool.Submit(task)

	return nil
}
