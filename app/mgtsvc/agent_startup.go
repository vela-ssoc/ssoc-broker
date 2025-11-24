package mgtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"gorm.io/gen/field"
)

func (biz *agentService) ReloadStartup(_ context.Context, mid int64) error {
	task := &startupTask{biz: biz, mid: mid}
	biz.pool.Go(task.Run)
	return nil
}

type startupTask struct {
	biz *agentService
	mid int64
}

func (st *startupTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	mid := st.mid
	startup, err := st.getStartup(ctx, mid)
	if err != nil {
		return
	}

	path := "/api/v1/agent/startup"
	err = st.biz.lnk.Oneway(ctx, mid, path, startup)

	tbl := st.biz.qry.Startup
	var assigns []field.AssignExpr
	if err != nil {
		assigns = append(assigns, tbl.Failed.Value(true))
		assigns = append(assigns, tbl.Reason.Value(err.Error()))
	} else {
		assigns = append(assigns, tbl.Failed.Value(false))
		assigns = append(assigns, tbl.Reason.Value(""))
	}

	_, _ = tbl.WithContext(ctx).
		Where(tbl.ID.Eq(mid)).
		UpdateSimple(assigns...)
}

func (st *startupTask) getStartup(ctx context.Context, mid int64) (*model.StartupFallback, error) {
	{
		tbl := st.biz.qry.Startup
		dao := tbl.WithContext(ctx)
		dat, err := dao.Where(tbl.ID.Eq(mid)).First()
		if err == nil && dat.Logger != nil {
			ret := &model.StartupFallback{
				ID:        mid,
				Logger:    *dat.Logger,
				CreatedAt: dat.CreatedAt,
				UpdatedAt: dat.UpdatedAt,
			}
			return ret, nil
		}
	}

	tbl := st.biz.qry.StartupFallback
	dao := tbl.WithContext(ctx)
	ret, err := dao.Order(tbl.ID.Desc()).First()
	if err != nil {
		return nil, err
	}
	ret.ID = mid

	return ret, nil
}
