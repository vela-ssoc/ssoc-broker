package shipx

import (
	"github.com/NVIDIA/gontainer/v2"
	"github.com/xgfone/ship/v5"
)

type HTTPRegister interface {
	RegisterHTTP(g RouteGroup) error
}

type RouteGroup struct {
	ManagerV3 *ship.RouteGroupBuilder // 供中心端调用
	AgentV3   *ship.RouteGroupBuilder // 供探针调用
	Expose    *ship.Ship              // 开放接口
}

func (sr RouteGroup) RegisterHTTP(handlers gontainer.Multiple[HTTPRegister]) error {
	for _, h := range handlers {
		if err := h.RegisterHTTP(sr); err != nil {
			return err
		}
	}

	return nil
}
