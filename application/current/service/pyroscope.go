package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"sync"

	"github.com/grafana/pyroscope-go"
	"github.com/vela-ssoc/ssoc-common/datalayer/model"
	"github.com/vela-ssoc/ssoc-common/datalayer/query"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/memcache"
	"github.com/vela-ssoc/ssoc-proto/muxproto"
)

type Pyroscope struct {
	this *model.Broker
	qry  *query.Query
	log  *slog.Logger
	mem  memcache.Cache[*model.Pyroscope, error]
	mtx  sync.Mutex // 防止并发启动
	prf  *pyroscope.Profiler
}

func NewPyroscope(this *model.Broker, qry *query.Query, log *slog.Logger) *Pyroscope {
	py := &Pyroscope{this: this, qry: qry, log: log}
	py.mem = memcache.NewCache(py.enabled)

	return py
}

func (py *Pyroscope) Start(ctx context.Context) error {
	py.mtx.Lock()
	defer py.mtx.Unlock()

	if py.prf != nil { // 已经启动成功了
		return nil
	}

	dat, err := py.Enabled(ctx)
	if err != nil {
		return err
	}

	strID := strconv.FormatInt(py.this.ID, 10)
	profileTypes := []pyroscope.ProfileType{
		pyroscope.ProfileCPU,
		pyroscope.ProfileInuseObjects,
		pyroscope.ProfileAllocObjects,
		pyroscope.ProfileInuseSpace,
		pyroscope.ProfileAllocSpace,
		pyroscope.ProfileGoroutines,
		pyroscope.ProfileMutexCount,
		pyroscope.ProfileMutexDuration,
		pyroscope.ProfileBlockCount,
		pyroscope.ProfileBlockDuration,
	}

	log := logger.NewFormat(py.log.Handler(), 5)
	cfg := pyroscope.Config{
		ApplicationName:   muxproto.BrokerDomain,
		Tags:              map[string]string{"instance": strID},
		ServerAddress:     dat.URL,
		BasicAuthUser:     dat.Username,
		BasicAuthPassword: dat.Password,
		Logger:            log,
		ProfileTypes:      profileTypes,
		HTTPHeaders:       dat.Header,
	}
	prf, err := pyroscope.Start(cfg)
	if err != nil {
		return err
	}
	py.prf = prf

	return nil
}

func (py *Pyroscope) Stop() error {
	py.mtx.Lock()
	prf := py.prf
	py.prf = nil
	py.mtx.Unlock()

	if prf != nil {
		return prf.Stop()
	}

	return nil
}

func (py *Pyroscope) Enabled(ctx context.Context) (*model.Pyroscope, error) {
	return py.mem.Load(ctx)
}

func (py *Pyroscope) enabled(ctx context.Context) (*model.Pyroscope, error) {
	tbl := py.qry.Pyroscope
	dao := tbl.WithContext(ctx)

	dat, err := dao.Where(tbl.Enabled.Is(true)).First()
	if err != nil {
		return nil, fmt.Errorf("查询启用的 pyroscope 配置出错: %w", err)
	}

	return dat, nil
}
