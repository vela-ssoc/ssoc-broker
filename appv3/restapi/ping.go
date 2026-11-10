package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type Ping struct{}

func NewPing() *Ping {
	return new(Ping)
}

func (p *Ping) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/ping").GET(p.ping)

	return nil
}

func (p *Ping) ping(c *ship.Context) error {
	return c.NoContent(http.StatusNoContent)
}
