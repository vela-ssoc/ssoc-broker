package restapi

import (
	"net/http/pprof"

	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type Pprof struct{}

func NewPprof() *Pprof {
	return new(Pprof)
}

func (prof *Pprof) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Route("/pprof/index").GET(prof.index)
	g.ManagerV3.Route("/pprof/cmdline").GET(prof.cmdline)
	g.ManagerV3.Route("/pprof/profile").GET(prof.profile)
	g.ManagerV3.Route("/pprof/symbol").GET(prof.symbol).POST(prof.symbol)
	g.ManagerV3.Route("/pprof/trace").GET(prof.trace)
	g.ManagerV3.Route("/pprof/:name").GET(prof.name)

	return nil
}

func (*Pprof) index(c *ship.Context) error {
	pprof.Index(c.Response(), c.Request())
	return nil
}

func (*Pprof) cmdline(c *ship.Context) error {
	pprof.Cmdline(c.Response(), c.Request())
	return nil
}

func (*Pprof) profile(c *ship.Context) error {
	pprof.Profile(c.Response(), c.Request())
	return nil
}

func (*Pprof) symbol(c *ship.Context) error {
	pprof.Symbol(c.Response(), c.Request())
	return nil
}

func (*Pprof) trace(c *ship.Context) error {
	pprof.Trace(c.Response(), c.Request())
	return nil
}

func (*Pprof) name(c *ship.Context) error {
	path := c.Param("name")
	pprof.Handler(path).ServeHTTP(c.Response(), c.Request())
	return nil
}
