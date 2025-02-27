package mgtapi

import (
	"os"

	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/xgfone/ship/v5"
)

func NewSystem() *System {
	return &System{}
}

type System struct{}

func (sys *System) Route(r *ship.RouteGroupBuilder) {
	r.Route(accord.PathElasticReset).
		Data(route.Named("退出程序")).POST(sys.exit)
}

func (sys *System) exit(_ *ship.Context) error {
	os.Exit(0)
	return nil
}
