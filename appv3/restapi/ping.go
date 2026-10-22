package restapi

import (
	"crypto/rand"
	"io"
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type Ping struct{}

func NewPing() *Ping {
	return new(Ping)
}

func (p *Ping) RegisterHTTP(g shipx.RouteGroup) error {
	g.AgentV3.Route("/ping").GET(p.ping)
	g.ManagerV3.Route("/ping").GET(p.ping)
	g.ManagerV3.Route("/chunk").GET(p.chunk)

	return nil
}

func (p *Ping) ping(c *ship.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (p *Ping) chunk(c *ship.Context) error {
	rd := io.LimitReader(rand.Reader, 512*1024)
	return c.Stream(http.StatusOK, ship.MIMEOctetStream, rd)
}
