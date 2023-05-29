package mgtapi

import (
	"net/http"

	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/app/service"
	"github.com/xgfone/ship/v5"
)

func Into(svc service.IntoService) route.Router {
	return &intoREST{svc: svc}
}

type intoREST struct {
	svc service.IntoService
}

func (rest *intoREST) Route(r *ship.RouteGroupBuilder) {
	methods := []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace,
		ship.PROPFIND, "LOCK", "MKCOL", "PROPPATCH", "COPY", "MOVE", "UNLOCK",
	}
	r.Route("/arr/*path").Method(rest.ARR, methods...)
}

func (rest *intoREST) ARR(c *ship.Context) error {
	r := c.Request()
	ctx := r.Context()
	sid := r.Header.Get("X-Node-Id")
	r.URL.Host = sid
	r.URL.Scheme = "http"

	return rest.svc.ARR(ctx, c.Response(), r)
}

func (rest *intoREST) AWS(c *ship.Context) error {
	return nil
}
