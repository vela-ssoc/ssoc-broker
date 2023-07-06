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
	r.Route(accord.PathTaskSync).Data(route.Named("节点配置同步")).POST(rest.Sync)
	r.Route(accord.PathTaskLoad).Data(route.Named("节点配置重启并同步")).POST(rest.Load)
	r.Route(accord.PathTaskTable).Data(route.Named("扫表并同步各个节点配置")).POST(rest.Table)
	r.Route(accord.PathStartup).Data(route.Named("startup 配置同步")).POST(rest.Startup)
}

// Sync 即：向指定 agent 同步配置
func (rest *taskREST) Sync(c *ship.Context) error {
	var req accord.TaskSyncRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.Sync(ctx, req.MinionID)
}

// Load 向指定节点发送指定配置（节点会重新加载指定配置）
func (rest *taskREST) Load(c *ship.Context) error {
	var req accord.TaskLoadRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.Load(ctx, req.MinionID, req.SubstanceID)
}

// Table 向指定节点发送指定配置（节点会重新加载指定配置）
func (rest *taskREST) Table(c *ship.Context) error {
	var req accord.TaskTable
	if err := c.Bind(&req); err != nil {
		return err
	}

	c.Infof("扫表同步节点配置")
	ctx := c.Request().Context()

	return rest.svc.Table(ctx, req.TaskID)
}

func (rest *taskREST) Startup(c *ship.Context) error {
	var req accord.Startup
	if err := c.Bind(&req); err != nil {
		return err
	}

	c.Infof("扫表同步节点配置")
	ctx := c.Request().Context()

	return rest.svc.Startup(ctx, req.ID)
}
