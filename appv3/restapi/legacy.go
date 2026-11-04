package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	"github.com/xgfone/ship/v5"
)

type Legacy struct {
	next muxproto.V1Accepter
}

func NewLegacy(next muxproto.V1Accepter) *Legacy {
	return &Legacy{next: next}
}

func (lg *Legacy) RegisterHTTP(g shipx.RouteGroup) error {
	g.ExposeV1.Group.Route("/minion").CONNECT(lg.endpoint) // 上一版本通道上线。

	return nil
}

func (lg *Legacy) endpoint(c *ship.Context) error {
	w, r := c.ResponseWriter(), c.Request()
	return lg.next.AcceptTCP(w, r)
}
