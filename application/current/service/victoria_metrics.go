package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vela-ssoc/ssoc-common/memcache"
	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
)

type VictoriaMetricsConfig struct {
	db  repository.Database
	log *slog.Logger
	mem memcache.Cache[*model.VictoriaMetricsConfig, error]
}

func NewVictoriaMetricsConfig(db repository.Database, log *slog.Logger) *VictoriaMetricsConfig {
	vm := &VictoriaMetricsConfig{db: db, log: log}
	vm.mem = memcache.NewCache(vm.enabled)

	return vm
}

func (vm *VictoriaMetricsConfig) LoadConfig(ctx context.Context) (string, *metrics.PushOptions, error) {
	dat, err := vm.mem.Load(ctx)
	if err != nil {
		return "", nil, err
	}

	headers := dat.Headers.Lines()
	if dat.Username != "" || dat.Password != "" {
		auth := dat.Username + ":" + dat.Password
		basic := base64.StdEncoding.EncodeToString([]byte(auth))
		pair := "Authorization: Basic " + basic
		headers = append(headers, pair)
	}

	opts := &metrics.PushOptions{
		Headers: headers,
		Method:  dat.Method,
	}

	return dat.URL, opts, nil
}

func (vm *VictoriaMetricsConfig) enabled(ctx context.Context) (*model.VictoriaMetricsConfig, error) {
	coll := vm.db.VictoriaMetricsConfig()
	cfg, err := coll.Enabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询 victoria-metrics 配置出错: %w", err)
	}

	return cfg, nil
}
