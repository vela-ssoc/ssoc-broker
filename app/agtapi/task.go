package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/xgfone/ship/v5"
)

func Task() route.Router {
	return &taskREST{}
}

type taskREST struct{}

func (rest *taskREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/task/status").
		Data(route.Named("上报任务运行状态")).POST(rest.Status)
}

func (rest *taskREST) Status(c *ship.Context) error {
	var req param.TaskReport
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	// 删除旧数据
	mid := inf.Issue().ID
	inet := inf.Inet().String()
	dats := req.ToModels(mid, inet)

	tbl := query.MinionTask
	_, _ = tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()
	if len(dats) != 0 {
		_ = tbl.WithContext(ctx).Create(dats...)
	}

	return nil
}
