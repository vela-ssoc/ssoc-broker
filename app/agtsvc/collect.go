package agtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"gorm.io/gorm/clause"
)

type CollectService interface {
	Sysinfo(info *model.SysInfo) error
	AccountFull(mid int64, dats []*model.MinionAccount) error
}

func Collect() CollectService {
	return &collectService{
		pool: gopool.New(30, 1024, time.Minute),
	}
}

type collectService struct {
	pool gopool.Executor
}

func (biz *collectService) Sysinfo(info *model.SysInfo) error {
	fn := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = query.SysInfo.WithContext(ctx).Save(info)
	}
	biz.pool.Execute(fn)
	return nil
}

func (biz *collectService) AccountFull(mid int64, dats []*model.MinionAccount) error {
	fn := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tbl := query.MinionAccount
		_, _ = tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()

		if len(dats) != 0 {
			_ = tbl.WithContext(ctx).
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(dats...)
		}
	}
	biz.pool.Execute(fn)
	return nil
}
