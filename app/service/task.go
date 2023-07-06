package service

import (
	"context"

	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/logback"
)

type TaskService interface {
	Sync(ctx context.Context, mid int64) error
	Load(ctx context.Context, mid, sid int64) error
	Table(ctx context.Context, tid int64) error
	Startup(ctx context.Context, id int64) error
}

func Task(lnk mlink.Linker, pool gopool.Executor, slog logback.Logger) TaskService {
	bid := lnk.Link().Ident().ID
	return &taskService{
		bid:  bid,
		lnk:  lnk,
		pool: pool,
		slog: slog,
	}
}

type taskService struct {
	bid  int64 // 该 broker 的 ID
	lnk  mlink.Linker
	pool gopool.Executor
	slog logback.Logger
}

func (biz *taskService) Sync(ctx context.Context, mid int64) error {
	task := subtask.SyncTask(biz.lnk, mid, biz.slog)
	biz.pool.Submit(task)
	return nil
}

func (biz *taskService) Load(ctx context.Context, mid, sid int64) error {
	// 查询配置信息
	tbl := query.Substance
	sub, err := tbl.WithContext(ctx).
		Where(tbl.ID.Eq(sid)).
		First()
	if err != nil {
		return err
	}
	// 查询节点信息
	monTbl := query.Minion
	mon, err := monTbl.WithContext(ctx).
		Select(monTbl.ID, monTbl.Status, monTbl.Unload).
		Where(monTbl.ID.Eq(mid)).
		First()
	if err != nil {
		return err
	}
	status := mon.Status
	if mon.Unload || status == model.MSDelete || status == model.MSInactive {
		return nil
	}

	task := subtask.DiffTask(biz.lnk, sub, mid, biz.slog)
	biz.pool.Submit(task)

	return nil
}

func (biz *taskService) Table(ctx context.Context, tid int64) error {
	task := subtask.Table(biz.lnk, biz.bid, tid, biz.pool, biz.slog)
	biz.pool.Submit(task)
	return nil
}

func (biz *taskService) Startup(ctx context.Context, mid int64) error {
	tbl := query.Startup
	dat, err := tbl.WithContext(ctx).Where(tbl.ID.Eq(mid)).First()
	if err != nil {
		return err
	}

	path := "/api/v1/agent/startup"

	return biz.lnk.Oneway(mid, path, dat)
}
