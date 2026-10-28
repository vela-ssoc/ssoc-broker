package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/response"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type ItselfRoute struct {
	agt  *ship.Ship
	mana *ship.Ship
	exp  *ship.Ship
}

func NewItselfRoute() *ItselfRoute {
	return new(ItselfRoute)
}

func (rt *ItselfRoute) RegisterHTTP(g shipx.RouteGroup) error {
	rt.agt = g.AgentV3.Group.Ship()
	rt.mana = g.ManagerV3.Group.Ship()
	rt.exp = g.ExposeV3.Group.Ship()

	g.ManagerV3.Group.Route("/itself/route/agents").GET(rt.agents)     // 探针侧接口
	g.ManagerV3.Group.Route("/itself/route/managers").GET(rt.managers) // 中心端侧接口
	g.ManagerV3.Group.Route("/itself/route/exposes").GET(rt.exposes)   // 用户侧接口

	return nil
}

func (rt *ItselfRoute) agents(c *ship.Context) error {
	ret := rt.foreach(rt.agt)
	return c.JSON(http.StatusOK, ret)
}

func (rt *ItselfRoute) managers(c *ship.Context) error {
	ret := rt.foreach(rt.mana)
	return c.JSON(http.StatusOK, ret)
}

func (rt *ItselfRoute) exposes(c *ship.Context) error {
	ret := rt.foreach(rt.exp)
	return c.JSON(http.StatusOK, ret)
}

func (*ItselfRoute) foreach(sh *ship.Ship) response.Records[response.RouteInfo] {
	dats := make([]response.RouteInfo, 0, 50)
	if sh != nil && sh.Router != nil {
		sh.Router.Range(func(_, path, method string, _ any) {
			item := response.RouteInfo{Method: method, Path: path}
			dats = append(dats, item)
		})
	}

	return response.NewRecords(dats)
}
