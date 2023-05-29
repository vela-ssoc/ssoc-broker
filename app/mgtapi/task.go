package mgtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/app/service"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/xgfone/ship/v5"
)

func Task(svc service.TaskService) route.Router {
	return &taskREST{svc: svc}
}

type taskREST struct {
	svc service.TaskService
}

func (rest *taskREST) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathTaskSync).POST(rest.Sync)
	r.Route(accord.PathTaskLoad).POST(rest.Load)
}

// Sync 即：向指定 agent 同步配置
func (rest *taskREST) Sync(c *ship.Context) error {
	var req accord.TaskSyncRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.Sync(ctx, req.MinionID, req.Inet)
}

// Load 向指定节点发送指定配置（节点会重新加载指定配置）
func (rest *taskREST) Load(c *ship.Context) error {
	var req accord.TaskLoadRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.Load(ctx, req.MinionID, req.SubstanceID, req.Inet)
}
