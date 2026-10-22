package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
	"golang.org/x/net/webdav"
)

func NewWebDAV(dir string) *WebDAV {
	return &WebDAV{
		webDAV: &webdav.Handler{
			FileSystem: webdav.Dir(dir),
			LockSystem: webdav.NewMemLS(),
		},
	}
}

type WebDAV struct {
	webDAV *webdav.Handler
}

func (d *WebDAV) RegisterHTTP(g shipx.RouteGroup) error {
	d.webDAV.Prefix = "/api/v3/webdav"
	methods := []string{
		http.MethodOptions, http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodDelete,
		"LOCK", "UNLOCK", "PROPFIND", "PROPPATCH", "MKCOL", "COPY", "MOVE",
	}

	g.ManagerV3.Route("/webdav").Method(d.browse, methods...)
	g.ManagerV3.Route("/webdav/*").Method(d.browse, methods...)

	return nil
}

func (d *WebDAV) browse(c *ship.Context) error {
	w, r := c.Response(), c.Request()
	d.webDAV.ServeHTTP(w, r)
	return nil
}
