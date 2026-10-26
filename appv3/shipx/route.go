package shipx

import (
	"github.com/NVIDIA/gontainer/v2"
	"github.com/xgfone/ship/v5"
)

type HTTPRegistrar interface {
	RegisterHTTP(g RouteGroup) error
}

type RouteEndpoint struct {
	Path  string                  // 路径前缀，例如 /api/v3
	Group *ship.RouteGroupBuilder // 路由注册分组
}

type RouteGroup struct {
	ExposeV1  RouteEndpoint
	ExposeV3  RouteEndpoint
	ManagerV1 RouteEndpoint
	ManagerV3 RouteEndpoint
	AgentV1   RouteEndpoint
	AgentV3   RouteEndpoint
}

func (g RouteGroup) RegisterRoutes(hs gontainer.Multiple[HTTPRegistrar]) error {
	for _, h := range hs {
		if err := h.RegisterHTTP(g); err != nil {
			return err
		}
	}

	return nil
}
