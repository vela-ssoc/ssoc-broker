package mrestapi

import (
	"log/slog"

	"github.com/vela-ssoc/vela-broker/appv2/manager/mrequest"
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
	r.Route("/system/update").POST(sys.update)
	return nil
}

func (sys *System) exit(_ *ship.Context) error {
	sys.svc.Exit()
	return nil
}

func (sys *System) update(c *ship.Context) error {
	req := new(mrequest.SystemUpdate)
	if err := c.Bind(req); err != nil {
		return err
	}

	err := sys.svc.Update(req.Semver)
	if err != nil {
		c.Warnf("升级错误", slog.Any("error", err))
	}

	return nil
}
