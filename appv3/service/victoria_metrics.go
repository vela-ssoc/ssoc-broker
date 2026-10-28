package service

import (
	"context"
	"log/slog"
	"net/url"
	"path"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vela-ssoc/ssoc-broker/errcode"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/memcache"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type VictoriaMetrics struct {
	db    polydb.Sourcer
	log   *slog.Logger
	label string
	mem   *memcache.TTL[*model.VictoriaMetricsConfig]
}

func NewVictoriaMetrics(db polydb.Sourcer, cfg model.BrokerConfig, log *slog.Logger) *VictoriaMetrics {
	vmc := &VictoriaMetrics{
		db:    db,
		log:   log,
		label: `appname="ssoc-broker",instance=` + strconv.Quote(cfg.Slug), // 固定 Label
	}
	vmc.mem = memcache.NewTTL(time.Minute, vmc.Get)

	return vmc
}

func (vmc *VictoriaMetrics) Get(ctx context.Context) (*model.VictoriaMetricsConfig, error) {
	mdb := vmc.db.MDB()
	coll := mdb.VictoriaMetricsConfig()

	return coll.FindOne(ctx, bson.D{})
}

func (vmc *VictoriaMetrics) Forget() {
	vmc.mem.Forget()
}

func (vmc *VictoriaMetrics) LoadVictoriaPushConfig(ctx context.Context) (pushURL string, opts *metrics.PushOptions, err error) {
	pu, headers, data, err := vmc.load(ctx)
	if err != nil {
		return "", nil, err
	}

	pre := data.InsertPrefixPath()
	suffix := path.Join(pre, "/api/v1/import/prometheus")
	pu = pu.JoinPath(suffix)
	opts = &metrics.PushOptions{ExtraLabels: vmc.label, Headers: headers.Array()}

	return pu.String(), opts, nil
}

func (vmc *VictoriaMetrics) load(ctx context.Context) (*url.URL, model.HTTPHeader, *model.VictoriaMetricsConfig, error) {
	dat, err := vmc.mem.Load(ctx)
	if err != nil {
		return nil, nil, nil, errcode.ErrVictoriaNotConfigured
	}
	if !dat.Enabled {
		return nil, nil, nil, errcode.ErrVictoriaDisabled
	}
	addr, header, err := dat.Target()
	if err != nil {
		vmc.log.Error("VictoriaMetrics 配置项错误", "config", dat, "err", err)
		return nil, nil, nil, err
	}

	return addr, header, dat, nil
}
