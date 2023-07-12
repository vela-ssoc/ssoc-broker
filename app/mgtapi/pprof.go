package mgtapi

import (
	"net/http"
	"net/http/pprof"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/xgfone/ship/v5"
)

func Pprof(lnk telecom.Linker) route.Router {
	return &pprofREST{
		lnk: lnk,
	}
}

type pprofREST struct {
	lnk telecom.Linker
}

func (rest *pprofREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/brr/debug/config").Data(route.Named("pprof-config")).GET(rest.Config)
	r.Route("/brr/debug/pprof").Data(route.Named("pprof-index")).GET(rest.Index)
	r.Route("/brr/debug/cmdline").Data(route.Named("pprof-cmdline")).GET(rest.Cmdline)
	r.Route("/brr/debug/profile").Data(route.Named("pprof-profile")).GET(rest.Profile)
	r.Route("/brr/debug/symbol").Data(route.Named("pprof-symbol")).GET(rest.Symbol)
	r.Route("/brr/debug/trace").Data(route.Named("pprof-trace")).GET(rest.Trace)
	r.Route("/brr/debug/*path").Data(route.Named("pprof-path")).GET(rest.Path)
}

func (rest *pprofREST) Config(c *ship.Context) error {
	hide := rest.lnk.Hide()
	ident := rest.lnk.Ident()
	issue := rest.lnk.Issue()

	res := &param.PprofConfig{
		Hide:  hide,
		Ident: ident,
		Issue: issue,
	}

	return c.JSON(http.StatusOK, res)
}

func (rest *pprofREST) Index(c *ship.Context) error {
	pprof.Index(c.Response(), c.Request())
	return nil
}

func (rest *pprofREST) Cmdline(c *ship.Context) error {
	pprof.Cmdline(c.Response(), c.Request())
	return nil
}

func (rest *pprofREST) Profile(c *ship.Context) error {
	pprof.Profile(c.Response(), c.Request())
	return nil
}

func (rest *pprofREST) Symbol(c *ship.Context) error {
	pprof.Symbol(c.Response(), c.Request())
	return nil
}

func (rest *pprofREST) Trace(c *ship.Context) error {
	pprof.Trace(c.Response(), c.Request())
	return nil
}

func (rest *pprofREST) Path(c *ship.Context) error {
	path := c.Param("path")
	pprof.Handler(path).ServeHTTP(c.Response(), c.Request())
	return nil
}
