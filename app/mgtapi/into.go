package mgtapi

import (
	"net/http"
	"strconv"

	"github.com/vela-ssoc/vela-broker/app/mgtsvc"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/xgfone/ship/v5"
)

func Into(svc mgtsvc.IntoService) route.Router {
	return &intoREST{svc: svc}
}

type intoREST struct {
	svc mgtsvc.IntoService
}

func (rest *intoREST) Route(r *ship.RouteGroupBuilder) {
	methods := []string{
		http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace,
		ship.PROPFIND, "LOCK", "MKCOL", "PROPPATCH", "COPY", "MOVE", "UNLOCK",
	}
	r.Route("/arr/*path").Data(route.Named("ARR 直接调用")).Method(rest.ARR, methods...)
	r.Route("/aws/*path").Data(route.Named("AWS 直接调用")).GET(rest.AWS)
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
	if !c.IsWebSocket() {
		return ship.ErrBadRequest
	}

	w, r := c.Response(), c.Request()
	ctx := r.Context()
	sid := r.Header.Get("X-Node-Id")
	id, _ := strconv.ParseInt(sid, 10, 64)

	return rest.svc.AWS(ctx, w, r, id)
}
