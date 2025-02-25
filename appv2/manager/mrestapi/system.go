package mrestapi

import (
	"github.com/vela-ssoc/vela-broker/appv2/manager/mservice"
	"github.com/xgfone/ship/v5"
)

func NewSystem(svc *mservice.System) *System {
	return &System{
		svc: svc,
	}
}

type System struct {
	svc *mservice.System
}

func (sys *System) BindRoute(r *ship.RouteGroupBuilder) error {
	r.Route("/system/exit").POST(sys.exit)
	return nil
}

func (sys *System) exit(_ *ship.Context) error {
	sys.svc.Exit()
	return nil
}
