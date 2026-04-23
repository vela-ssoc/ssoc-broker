package service

import (
	"context"
	"encoding/base64"
	"log/slog"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common/memcache"
)

type VictoriaMetricsConfig struct {
	qry         *query.Query
	log         *slog.Logger
	extraLabels string
	cache       *memcache.TTLCache[*model.VictoriaMetricsConfig]
}

func NewVictoriaMetricsConfig(qry *query.Query, extraLabels string, log *slog.Logger) *VictoriaMetricsConfig {
	vmc := &VictoriaMetricsConfig{
		qry:         qry,
		log:         log,
		extraLabels: extraLabels,
	}
	vmc.cache = memcache.NewTTLCache(time.Minute, vmc.slowLoad)

	return vmc
}

func (vmc *VictoriaMetricsConfig) LoadConfig(ctx context.Context) (string, *metrics.PushOptions, error) {
	cfg, err := vmc.cache.Load(ctx)
	if err != nil {
		return "", nil, err
	}

	opts := &metrics.PushOptions{
		ExtraLabels: vmc.extraLabels,
		Headers:     nil,
		Method:      cfg.Method,
	}
	if cfg.Username != "" || cfg.Password != "" {
		pair := cfg.Username + ":" + cfg.Password
		auth := base64.StdEncoding.EncodeToString([]byte(pair))
		opts.Headers = []string{"Authorization: Basic " + auth}
	}

	return cfg.URL, opts, nil
}

func (vmc *VictoriaMetricsConfig) slowLoad(ctx context.Context) (*model.VictoriaMetricsConfig, error) {
	tbl := vmc.qry.VictoriaMetricsConfig
	dao := tbl.WithContext(ctx)

	return dao.Where(tbl.Enabled.Is(true)).
		Order(tbl.UpdatedAt.Desc()).
		First()
}
