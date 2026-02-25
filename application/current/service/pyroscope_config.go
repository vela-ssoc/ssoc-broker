package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/grafana/pyroscope-go"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"github.com/vela-ssoc/ssoc-proto/muxproto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type PyroscopeConfig struct {
	db     repository.Database
	instID bson.ObjectID
	log    *slog.Logger
	mtx    sync.Mutex // 防止并发启动
	prf    *pyroscope.Profiler
}

func NewPyroscopeConfig(db repository.Database, instanceID bson.ObjectID, log *slog.Logger) *PyroscopeConfig {
	return &PyroscopeConfig{
		db:     db,
		instID: instanceID,
		log:    log,
	}
}

func (py *PyroscopeConfig) Start(ctx context.Context) error {
	py.mtx.Lock()
	defer py.mtx.Unlock()

	if py.prf != nil { // 已经启动成功了
		return nil
	}

	dat, err := py.enabled(ctx)
	if err != nil {
		return err
	}

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

	log := logger.NewFormat(py.log.Handler(), 6)
	cfg := pyroscope.Config{
		ApplicationName:   muxproto.BrokerDomain,
		Tags:              map[string]string{"instance": py.instID.Hex()},
		ServerAddress:     dat.URL,
		BasicAuthUser:     dat.Username,
		BasicAuthPassword: dat.Password,
		Logger:            log,
		ProfileTypes:      profileTypes,
		HTTPHeaders:       dat.Headers,
	}
	prf, err := pyroscope.Start(cfg)
	if err != nil {
		return err
	}
	py.prf = prf

	return nil
}

func (py *PyroscopeConfig) Close() error {
	py.mtx.Lock()
	defer py.mtx.Unlock()

	if py.prf == nil {
		return nil
	}

	err := py.prf.Stop()
	py.prf = nil

	return err
}

func (py *PyroscopeConfig) enabled(ctx context.Context) (*model.PyroscopeConfig, error) {
	coll := py.db.PyroscopeConfig()
	cfg, err := coll.Enabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询启用的 pyroscope 配置出错: %w", err)
	}

	return cfg, nil
}
