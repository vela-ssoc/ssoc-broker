package mgtapi

import (
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/app/service"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/xgfone/ship/v5"
)

func Agent(svc service.AgentService) route.Router {
	return &agentREST{
		svc: svc,
	}
}

type agentREST struct {
	svc service.AgentService
}

func (rest *agentREST) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathUpgrade).Data(route.Named("通知节点二进制升级")).POST(rest.Upgrade)
	r.Route(accord.PathStartup).Data(route.Named("通知节点 startup 更新")).POST(rest.Startup)
	r.Route(accord.PathCommand).Data(route.Named("通知节点执行命令")).POST(rest.Command)
}

func (rest *agentREST) Upgrade(c *ship.Context) error {
	var req accord.Upgrade
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.UpgradeID(ctx, req.ID)
}

func (rest *agentREST) Startup(c *ship.Context) error {
	var req accord.Startup
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return rest.svc.Startup(ctx, req.ID)
}

func (rest *agentREST) Command(c *ship.Context) error {
	var req accord.Command
	if err := c.Bind(&req); err != nil {
		return err
	}
	ctx := c.Request().Context()

	return rest.svc.Command(ctx, req.ID, req.Cmd)
}
