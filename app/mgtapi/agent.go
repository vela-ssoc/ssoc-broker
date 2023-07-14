package mgtapi

import (
	"github.com/vela-ssoc/vela-broker/app/mgtsvc"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/xgfone/ship/v5"
)

func Agent(svc mgtsvc.AgentService) route.Router {
	return &agentREST{
		svc: svc,
	}
}

type agentREST struct {
	svc mgtsvc.AgentService
}

func (rest *agentREST) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathUpgrade).Data(route.Named("通知节点二进制升级")).POST(rest.Upgrade)
	r.Route(accord.PathStartup).Data(route.Named("通知节点 startup 更新")).POST(rest.Startup)
	r.Route(accord.PathCommand).Data(route.Named("通知节点执行命令")).POST(rest.Command)
	r.Route(accord.PathTaskSync).Data(route.Named("通知节点同步配置")).POST(rest.RsyncTask)
	r.Route(accord.PathTaskLoad).Data(route.Named("通知节点重启配置")).POST(rest.ReloadTask)
	r.Route(accord.PathTaskTable).Data(route.Named("通知扫表任务")).POST(rest.TableTask)
}

func (rest *agentREST) Upgrade(c *ship.Context) error {
	var req accord.Upgrade
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.Upgrade(ctx, req.ID, req.Semver)
}

func (rest *agentREST) Startup(c *ship.Context) error {
	var req accord.Startup
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.ReloadStartup(ctx, req.ID)
}

func (rest *agentREST) Command(c *ship.Context) error {
	// resync restart upgrade offline
	var req accord.Command
	if err := c.Bind(&req); err != nil {
		return err
	}
	ctx := c.Request().Context()

	switch req.Cmd {
	case "resync":
		return rest.svc.RsyncTask(ctx, req.ID)
	case "upgrade":
		return rest.svc.Upgrade(ctx, req.ID, "")
	}

	return rest.svc.Command(ctx, req.ID, req.Cmd)
}

func (rest *agentREST) Offline(c *ship.Context) error {
	var req accord.IDs
	if err := c.Bind(&req); err != nil {
		return err
	}
	ctx := c.Request().Context()

	return rest.svc.Command(ctx, req.ID, "offline")
}

func (rest *agentREST) RsyncTask(c *ship.Context) error {
	var req accord.IDs
	if err := c.Bind(&req); err != nil {
		return err
	}
	ctx := c.Request().Context()

	return rest.svc.RsyncTask(ctx, req.ID)
}

func (rest *agentREST) TableTask(c *ship.Context) error {
	var req accord.TaskTable
	if err := c.Bind(&req); err != nil {
		return err
	}
	ctx := c.Request().Context()

	return rest.svc.TableTask(ctx, req.TaskID)
}

func (rest *agentREST) ReloadTask(c *ship.Context) error {
	var req accord.TaskLoadRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	ctx := c.Request().Context()

	return rest.svc.ReloadTask(ctx, req.MinionID, req.SubstanceID)
}
