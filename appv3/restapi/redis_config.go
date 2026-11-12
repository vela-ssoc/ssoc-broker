package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type RedisConfig struct {
	svc *service.RedisConfig
}

func NewRedisConfig(svc *service.RedisConfig) *RedisConfig {
	return &RedisConfig{svc: svc}
}

func (rc *RedisConfig) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/redis-config").DELETE(rc.forget) // 修改/新增 Redis 配置

	return nil
}

func (rc *RedisConfig) forget(c *ship.Context) error {
	rc.svc.Forget()
	return nil
}
