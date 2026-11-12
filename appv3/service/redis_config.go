package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/redis/go-redis/v9"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/memcache"
)

type RedisConfig struct {
	pdb polydb.Sourcer
	log *slog.Logger
	mem *memcache.Value[redis.UniversalClient]
}

func NewRedisConfig(pdb polydb.Sourcer, log *slog.Logger) *RedisConfig {
	rc := &RedisConfig{pdb: pdb, log: log}
	rc.mem = memcache.NewValue(rc.slowLoad)

	return rc
}

func (rc *RedisConfig) Forget() {
	if cli, _ := rc.mem.Forget(); cli != nil {
		_ = cli.Close()
	}
}

func (rc *RedisConfig) slowLoad(ctx context.Context) (redis.UniversalClient, error) {
	return nil, errors.ErrUnsupported
}
