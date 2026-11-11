package restapi

import (
	"net/http"
	"net/http/httputil"
	"path"

	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/httpship"
	"github.com/xgfone/ship/v5"
)

type AgentReverseProxy struct {
	rt  http.RoundTripper
	pxy *httputil.ReverseProxy
}

func NewAgentReverseProxy(rt http.RoundTripper) *AgentReverseProxy {
	pxy := &httputil.ReverseProxy{
		Transport: rt,
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetXForwarded()
		},
		ErrorHandler: httpship.ErrorHandle,
	}

	return &AgentReverseProxy{rt: rt, pxy: pxy}
}

func (arp *AgentReverseProxy) RegisterHTTP(g shipx.RouteGroup) error {
	methods := []string{
		http.MethodOptions, http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete,
		"QUERY", "LOCK", "UNLOCK", "PROPFIND", "PROPPATCH", "MKCOL", "COPY", "MOVE",
	}

	g.ManagerV1.Group.Route("/arr").Method(arp.handle, methods...)
	g.ManagerV1.Group.Route("/arr/*path").Method(arp.handle, methods...)

	return nil
}

func (arp *AgentReverseProxy) handle(c *ship.Context) error {
	// 隧道内部走 http 协议
	w, r := httpship.Strip(c, "path")
	r.URL.Path = path.Join("/api/v1/arr", r.URL.Path) // v1 版本 agent 要加上 /api/v1/arr 前缀
	r.URL.Scheme = "http"
	r.URL.Host = r.Host
	arp.pxy.ServeHTTP(w, r)

	return nil
}
