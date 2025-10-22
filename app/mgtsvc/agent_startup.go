package mgtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mba/definition"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

func (biz *agentService) ReloadStartup(_ context.Context, mid int64) error {
	task := &startupTask{biz: biz, mid: mid}
	biz.pool.Go(task.Run)
	return nil
}

func (biz *agentService) findStartup(ctx context.Context, mid int64) (*definition.Startup, error) {
	tbl := biz.qry.Startup
	dat, err := tbl.WithContext(ctx).Where(tbl.ID.Eq(mid)).First()
	if err == gorm.ErrRecordNotFound {
		dat, err = biz.store.Startup(ctx)
	}
	if err != nil {
		return nil, err
	}

	ret := biz.convertStartup(dat)
	return ret, nil
}

func (*agentService) convertStartup(dat *model.Startup) *definition.Startup {
	logg := dat.Logger

	return &definition.Startup{
		Logger: definition.StartupLogger{
			Level:    logg.Level,
			Filename: logg.Filename,
			Console:  logg.Console,
			Format:   logg.Format,
			Caller:   logg.Caller,
			Skip:     logg.Skip,
		},
	}
}

type startupTask struct {
	biz *agentService
	mid int64
}

func (st *startupTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	mid := st.mid
	startup, err := st.biz.findStartup(ctx, mid)
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
