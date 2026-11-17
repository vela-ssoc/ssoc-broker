package restapi

import (
	"errors"

	"github.com/vela-ssoc/ssoc-broker/agentask"
	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type AgentTask struct {
	cli agentask.Client
}

func NewAgentTask(cli agentask.Client) *AgentTask {
	return &AgentTask{cli: cli}
}

func (at *AgentTask) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/agent-task/resync").POST(at.resync) // 同步配置状态
	g.ManagerV3.Group.Route("/agent-task/reload").POST(at.reload) // 重新加载指定配置

	return nil
}

func (at *AgentTask) resync(c *ship.Context) error {
	var param request.Int64IDs
	if err := c.Bind(&param); err != nil {
		return err
	}
	ctx := c.Request().Context()

	var errs []error
	for _, id := range param.ID {
		err1 := at.cli.ResyncTasks(ctx, id, 3)
		errs = append(errs, err1)
	}

	return errors.Join(errs...)
}

func (at *AgentTask) reload(c *ship.Context) error {
	var param request.AgentTaskReload
	if err := c.Bind(&param); err != nil {
		return err
	}
	ctx := c.Request().Context()

	return at.cli.ReloadTask(ctx, param.ID, param.SubstanceID)
}
