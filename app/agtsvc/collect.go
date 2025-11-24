package agtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common-mb/gopool"
	"gorm.io/gorm/clause"
)

type CollectService interface {
	Sysinfo(info *model.SysInfo) error
	AccountFull(mid int64, dats []*model.MinionAccount) error
}

func NewCollect(qry *query.Query) CollectService {
	return &collectService{
		qry:  qry,
		pool: gopool.New(1024),
	}
}

type collectService struct {
	qry  *query.Query
	pool gopool.Pool
}

func (biz *collectService) Sysinfo(info *model.SysInfo) error {
	fn := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 新增/更新 sysinfo 表
		{
			tbl := biz.qry.SysInfo
			dao := tbl.WithContext(ctx)
			_ = dao.Save(info)
		}

		// 更新 minion 表
		{
			tbl := biz.qry.Minion
			dao := tbl.WithContext(ctx)
			_, _ = dao.Where(tbl.ID.Eq(info.ID)).
				UpdateSimple(tbl.OSRelease.Value(info.Release))
		}
	}
	biz.pool.Go(fn)
	return nil
}

func (biz *collectService) AccountFull(mid int64, dats []*model.MinionAccount) error {
	fn := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tbl := biz.qry.MinionAccount
		_, _ = tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()

		if len(dats) != 0 {
			_ = tbl.WithContext(ctx).
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(dats...)
		}
	}
	biz.pool.Go(fn)
	return nil
}
