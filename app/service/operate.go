package service

import (
	"context"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
	"github.com/vela-ssoc/vela-manager/errcode"
	"gorm.io/gorm/clause"
)

type OperateService interface {
	Update(ctx context.Context, mid int64, req *param.TagRequest) error
}

func Operate(
	lnk mlink.Linker,
	comp subtask.Comparer,
	pool taskpool.Executor,
	slog logback.Logger,
) OperateService {
	return &operateService{
		lnk:  lnk,
		comp: comp,
		pool: pool,
		slog: slog,
	}
}

type operateService struct {
	lnk  mlink.Linker
	comp subtask.Comparer
	pool taskpool.Executor
	slog logback.Logger
}

func (biz *operateService) Update(ctx context.Context, mid int64, req *param.TagRequest) error {
	if req.Empty() {
		return nil
	}

	monTbl := query.Minion
	mon, err := monTbl.WithContext(ctx).
		Select(monTbl.Status, monTbl.BrokerID, monTbl.Inet).
		Where(monTbl.ID.Eq(mid)).
		First()
	if err != nil {
		return err
	}
	if mon.Status == model.MSDelete {
		return errcode.ErrNodeStatus
	}

	tbl := query.MinionTag
	// 查询现有的 tags
	olds, err := tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Find()
	if err != nil {
		return err
	}
	news := model.MinionTags(olds).Minion(mid, req.Del, req.Add)
	err = query.Q.Transaction(func(tx *query.Query) error {
		table := tx.WithContext(ctx).MinionTag
		if _, exx := table.Where(tbl.MinionID.Eq(mid)).
			Delete(); exx != nil || len(news) == 0 {
			return exx
		}
		return table.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(news, 100)
	})

	if err == nil {
		task := subtask.SyncTask(biz.lnk, biz.comp, mid, mon.Inet, biz.slog)
		biz.pool.Submit(task)
	}

	return err
}
