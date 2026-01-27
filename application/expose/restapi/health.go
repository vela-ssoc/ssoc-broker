package restapi

import (
	"net/http"

	"github.com/xgfone/ship/v5"
)

type Health struct {
}

func NewHealth() *Health {
	return &Health{}
}

func (h *Health) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/health/ping").GET(h.ping)
	return nil
}

func (h *Health) ping(c *ship.Context) error {
	return c.NoContent(http.StatusNoContent)
}
