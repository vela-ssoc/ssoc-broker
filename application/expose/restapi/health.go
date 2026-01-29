package restapi

import (
	"net/http"
	"net/http/httputil"

	"github.com/vela-ssoc/ssoc-proto/muxproto"
	"github.com/vela-ssoc/ssoc-proto/muxtool"
	"github.com/xgfone/ship/v5"
)

type Health struct {
	prx *httputil.ReverseProxy
}

func NewHealth(cli muxtool.Client) *Health {
	destURL := muxproto.ToManagerURL("/")
	prx := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetXForwarded()
			pr.SetURL(destURL)
		},
		Transport: cli.Transport(),
	}

	return &Health{
		prx: prx,
	}
}

func (h *Health) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/health/ping").GET(h.ping)
	rgb.Route("/download").GET(h.download)
	return nil
}

func (h *Health) ping(c *ship.Context) error {
	return c.NoContent(http.StatusNoContent)
}

func (h *Health) download(c *ship.Context) error {
	h.prx.ServeHTTP(c, c.Request())
	return nil
}
