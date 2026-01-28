package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vela-ssoc/ssoc-common/datalayer/model"
	"github.com/vela-ssoc/ssoc-common/datalayer/query"
	"github.com/vela-ssoc/ssoc-common/memcache"
)

type VictoriaMetrics struct {
	qry *query.Query
	log *slog.Logger
	mem memcache.Cache[*model.VictoriaMetrics, error]
}

func NewVictoriaMetrics(qry *query.Query, log *slog.Logger) *VictoriaMetrics {
	vm := &VictoriaMetrics{qry: qry, log: log}
	vm.mem = memcache.NewCache(vm.enabled)

	return vm
}

func (vm *VictoriaMetrics) Load(ctx context.Context) (string, *metrics.PushOptions, error) {
	dat, err := vm.mem.Load(ctx)
	if err != nil {
		return "", nil, err
	}

	opts := &metrics.PushOptions{
		Headers: dat.Header,
		Method:  dat.Method,
	}

	return dat.URL, nil, nil
}

func (vm *VictoriaMetrics) enabled(ctx context.Context) (*model.VictoriaMetrics, error) {
	tbl := vm.qry.VictoriaMetrics
	dao := tbl.WithContext(ctx)

	dat, err := dao.Where(tbl.Enabled.Is(true)).First()
	if err != nil {
		return nil, fmt.Errorf("查询 victoria-metrics 配置出错: %w", err)
	}

	return dat, nil
}
