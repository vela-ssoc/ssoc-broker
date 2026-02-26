package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/magnetde/slog-loki"
	"github.com/vela-ssoc/ssoc-common/logger"
	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"github.com/vela-ssoc/ssoc-proto/muxproto"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type LokiConfig struct {
	db     repository.Database
	instID bson.ObjectID
	lmh    *logger.MultiHandler
	log    *slog.Logger
	mtx    sync.Mutex // 防止并发启动
	lok    *loki.Handler
}

func NewLokiConfig(db repository.Database, instanceID bson.ObjectID, lmh *logger.MultiHandler, log *slog.Logger) *LokiConfig {
	return &LokiConfig{
		db:     db,
		instID: instanceID,
		lmh:    lmh,
		log:    log,
	}
}

func (lc *LokiConfig) Start(ctx context.Context) error {
	lc.mtx.Lock()
	defer lc.mtx.Unlock()

	if lc.lok != nil { // 已经启动成功了
		return nil
	}

	cfg, err := lc.enabled(ctx)
	if err != nil {
		lc.log.Error("查询 loki 配置出错", "error", err)
		return err
	}

	instance := lc.instID.Hex()
	opts := []loki.Option{
		loki.WithName(muxproto.BrokerDomain),
		loki.WithLabel("instance", instance),
	}
	h := loki.NewHandler(cfg.URL, opts...)
	lc.lmh.Append(h)
	lc.lok = h

	return nil
}

func (lc *LokiConfig) Close() error {
	lc.mtx.Lock()
	defer lc.mtx.Unlock()

	if lc.lok == nil {
		return nil
	}

	lok := lc.lok
	lc.lok = nil
	lc.lmh.Remove(lok)

	return lok.Close()
}

func (lc *LokiConfig) enabled(ctx context.Context) (*model.LokiConfig, error) {
	coll := lc.db.LokiConfig()
	cfg, err := coll.Enabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询启用的 loki 配置出错: %w", err)
	}

	return cfg, nil
}
