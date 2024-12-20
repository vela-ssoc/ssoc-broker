package agtapi

import (
	"context"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/xgfone/ship/v5"
)

func Proxy(dc func(context.Context, string, string) (net.Conn, error)) route.Router {
	rawURL, _ := url.Parse("http://vtun/proxy/siem")
	siem := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(rawURL)
		},
		Transport: &http.Transport{
			DialContext: dc,
		},
	}

	return &proxyAPI{
		siem: siem,
	}
}

type proxyAPI struct {
	siem *httputil.ReverseProxy
}

func (api *proxyAPI) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/proxy/siem").Data(route.Ignore()).Any(api.siemFunc)
	r.Route("/broker/proxy/siem/*path").Data(route.Ignore()).Any(api.siemFunc)
}

// siem 通过 tunnel 方式代理向 SIEM 平台发起请求。
func (api *proxyAPI) siemFunc(c *ship.Context) error {
	path := "/" + c.Param("path")
	r, w := c.Request(), c.Response()
	r.URL.Path, r.RequestURI = path, path
	api.siem.ServeHTTP(w, r)

	return nil
}
