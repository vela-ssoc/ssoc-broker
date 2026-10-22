package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/response"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type Route struct {
	agt *ship.Ship
	brk *ship.Ship
	exp *ship.Ship
}

func NewRoute() *Route {
	return new(Route)
}

func (rt *Route) RegisterHTTP(g shipx.RouteGroup) error {
	rt.agt = g.AgentV3.Ship()
	rt.brk = g.ManagerV3.Ship()
	rt.exp = g.Expose

	g.ManagerV3.Route("/route/agents").GET(rt.agents)
	g.ManagerV3.Route("/route/brokers").GET(rt.brokers)
	g.ManagerV3.Route("/route/exposes").GET(rt.exposes)

	return nil
}

func (rt *Route) agents(c *ship.Context) error {
	ret := rt.foreach(rt.agt)
	return c.JSON(http.StatusOK, ret)
}

func (rt *Route) brokers(c *ship.Context) error {
	ret := rt.foreach(rt.brk)
	return c.JSON(http.StatusOK, ret)
}

func (rt *Route) exposes(c *ship.Context) error {
	ret := rt.foreach(rt.exp)
	return c.JSON(http.StatusOK, ret)
}

func (*Route) foreach(sh *ship.Ship) response.Records[response.RouteInfo] {
	dats := make([]response.RouteInfo, 0, 50)
	if sh != nil && sh.Router != nil {
		sh.Router.Range(func(_, path, method string, _ any) {
			item := response.RouteInfo{Method: method, Path: path}
			dats = append(dats, item)
		})
	}

	return response.NewRecords(dats)
}
