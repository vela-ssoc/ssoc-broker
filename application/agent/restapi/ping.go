package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-common/linkhub"
	"github.com/xgfone/ship/v5"
)

func NewPing() *Ping {
	return &Ping{}
}

type Ping struct{}

func (p Ping) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/ping").GET(p.pong)
	return nil
}

func (p Ping) pong(c *ship.Context) error {
	ctx := c.Request().Context()
	peer := linkhub.FromContext(ctx)
	agentID := peer.Info().ID
	c.Infof("节点发来了心跳消息", "agent_id", agentID)

	return c.NoContent(http.StatusOK)
}
