package restapi

import (
	"net/http"
	"time"

	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/response"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/selfexit"
	"github.com/xgfone/ship/v5"
)

type ItselfExit struct {
	self selfexit.Exit
}

func NewItselfExit() *ItselfExit {
	return new(ItselfExit)
}

func (et *ItselfExit) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/itself/exit").
		GET(et.deadline).
		POST(et.exit).
		DELETE(et.cancel)

	return nil
}

func (et *ItselfExit) exit(c *ship.Context) error {
	var param request.ItselfExit
	if err := c.Bind(&param); err != nil {
		return err
	}

	du := param.Duration()
	if du <= 0 {
		du = time.Second
	}

	c.Warnf("[程序退出] %s 后程序将退出执行", du)
	et.self.Exit(du, 0)

	return nil
}

func (et *ItselfExit) cancel(c *ship.Context) error {
	dead := et.self.Cancel()
	if !dead.IsZero() {
		c.Warnf("[程序退出] 程序将退出计划已取消")
	}

	return nil
}

func (et *ItselfExit) deadline(c *ship.Context) error {
	dead := et.self.Deadline()
	ret := response.NewData(dead)

	return c.JSON(http.StatusOK, ret)
}
