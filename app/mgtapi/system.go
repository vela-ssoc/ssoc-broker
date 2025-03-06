package mgtapi

import (
	"os"

	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/xgfone/ship/v5"
)

func NewSystem() *System {
	return &System{}
}

type System struct{}

func (sys *System) Route(r *ship.RouteGroupBuilder) {
	r.Route("/system/exit").
		Data(route.Named("退出程序")).POST(sys.exit)
	r.Route("/system/upgrade").
		Data(route.Named("程序升级")).POST(sys.upgrade)
}

func (sys *System) exit(_ *ship.Context) error {
	os.Exit(0)
	return nil
}

func (sys *System) upgrade(c *ship.Context) error {
	return nil
}
