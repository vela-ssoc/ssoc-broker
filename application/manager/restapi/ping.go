package restapi

import (
	"net/http"

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
	return c.NoContent(http.StatusOK)
}
