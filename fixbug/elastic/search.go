package elastic

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/vela-ssoc/vela-common-mba/netutil"
)

func NewSearch(cfg elastic.Configurer, cli netutil.HTTPClient) elastic.Searcher {
	ua := "elastic-ssoc-broker-" + runtime.GOOS + "-" + runtime.GOARCH
	return &searchClient{
		cfg:     cfg,
		ua:      ua,
		cli:     cli,
		timeout: 5 * time.Second,
	}
}

type searchClient struct {
	cfg     elastic.Configurer
	ua      string
	cli     netutil.HTTPClient
	timeout time.Duration
}

func (es *searchClient) ServeHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	h, err := es.cfg.Load(ctx)
	if err == nil {
		h.ServeHTTP(w, r)
	}
	return err
}
