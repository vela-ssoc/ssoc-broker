package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/memcache"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ReverseProxy struct {
	db      polydb.Sourcer
	log     *slog.Logger
	proxies *memcache.Map[string, *httputil.ReverseProxy]
}

func NewReverseProxy(db polydb.Sourcer, log *slog.Logger) *ReverseProxy {
	rp := &ReverseProxy{db: db, log: log}
	rp.proxies = memcache.NewMap(rp.findLocation)

	return rp
}

func (rp *ReverseProxy) Handle(ctx context.Context, location string, w http.ResponseWriter, r *http.Request) error {
	reverse, err := rp.proxies.Load(ctx, location)
	if err == nil {
		reverse.ServeHTTP(w, r)
	}

	return err
}

func (rp *ReverseProxy) Forget(locs []string) {
	if len(locs) == 0 {
		rp.proxies.ForgetAll()
		return
	}

	for _, loc := range locs {
		rp.proxies.Forget(loc)
	}
}

func (rp *ReverseProxy) findLocation(ctx context.Context, loc string) (*httputil.ReverseProxy, error) {
	coll := rp.db.MDB().ReverseProxy()
	dat, err := coll.FindOne(ctx, bson.M{"location": loc, "manager": true})
	if err != nil {
		return nil, err
	}
	target, err := url.Parse(dat.Upstream)
	if err != nil {
		return nil, err
	}

	pxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetXForwarded()
			pr.SetURL(target)
			for k, v := range dat.Header {
				pr.Out.Header.Set(k, v)
			}
		},
	}

	return pxy, nil
}
