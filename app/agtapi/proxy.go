package agtapi

import (
	"net/http/httputil"

	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/xgfone/ship/v5"
)

func Proxy(siemProxy *httputil.ReverseProxy) route.Router {
	return &proxyAPI{
		siemProxy: siemProxy,
	}
}

type proxyAPI struct {
	siemProxy *httputil.ReverseProxy
}

func (api *proxyAPI) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/proxy/siem").Data(route.Ignore()).Any(api.SIEM)
	r.Route("/broker/proxy/siem/*path").Data(route.Ignore()).Any(api.SIEM)
}

func (api *proxyAPI) SIEM(c *ship.Context) error {
	path := "/" + c.Param("path")

	r, w := c.Request(), c.Response()
	r.URL.Path = path
	r.RequestURI = path

	api.siemProxy.ServeHTTP(w, r)

	return nil
}
