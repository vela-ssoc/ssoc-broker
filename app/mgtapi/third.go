package mgtapi

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
	"github.com/xgfone/ship/v5"
)

func Third(lnk mlink.Linker, pool taskpool.Executor) route.Router {
	return &thirdREST{
		lnk:  lnk,
		pool: pool,
	}
}

type thirdREST struct {
	lnk  mlink.Linker
	pool taskpool.Executor
}

func (rest *thirdREST) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathThirdDiff).Data(route.Named("通知节点三方文件变动")).POST(rest.Diff)
}

func (rest *thirdREST) Diff(c *ship.Context) error {
	var req param.ThirdDiff
	if err := c.Bind(&req); err != nil {
		return err
	}

	task := subtask.ThirdDiff(rest.lnk, &req)
	rest.pool.Submit(task)

	return nil
}
