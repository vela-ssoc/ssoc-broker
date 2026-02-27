package restapi

import (
	"net/http"

	"github.com/xgfone/ship/v5"
)

type Heartbeat struct{}

func NewHeartbeat() *Heartbeat {
	return &Heartbeat{}
}

func (h *Heartbeat) RegisterRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/heartbeat").GET(h.ping)
	return nil
}

func (h *Heartbeat) ping(c *ship.Context) error {
	return c.NoContent(http.StatusNoContent)
}
