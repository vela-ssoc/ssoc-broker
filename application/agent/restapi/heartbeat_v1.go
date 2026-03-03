package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/application/agent/service"
	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/xgfone/ship/v5"
)

type HeartbeatV1 struct {
	svc *service.HeartbeatV1
}

func NewHeartbeatV1(svc *service.HeartbeatV1) *HeartbeatV1 {
	return &HeartbeatV1{
		svc: svc,
	}
}

func (h *HeartbeatV1) RegisterRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/minion/ping").POST(h.ping)

	return nil
}

func (h *HeartbeatV1) ping(c *ship.Context) error {
	ctx := c.Request().Context()
	peer := muxserver.FromContext(ctx)

	return h.svc.Ping(ctx, peer)
}
