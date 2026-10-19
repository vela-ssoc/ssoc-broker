package shipx

import (
	"github.com/NVIDIA/gontainer/v2"
	"github.com/xgfone/ship/v5"
)

type HTTPRegister interface {
	RegisterHTTP(g RouteGroup) error
}

type RouteGroup struct {
	AgentV3   *ship.RouteGroupBuilder
	ManagerV3 *ship.RouteGroupBuilder
}

func (sr RouteGroup) RegisterHTTP(handlers gontainer.Multiple[HTTPRegister]) error {
	for _, h := range handlers {
		if err := h.RegisterHTTP(sr); err != nil {
			return err
		}
	}

	return nil
}
