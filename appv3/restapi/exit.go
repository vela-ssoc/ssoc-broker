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

type Exit struct {
	self selfexit.Exit
}

func NewExit() *Exit {
	return new(Exit)
}

func (et *Exit) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Route("/soft/exit").
		GET(et.deadline).
		POST(et.exit).
		DELETE(et.cancel)

	return nil
}

func (et *Exit) exit(c *ship.Context) error {
	var param request.SoftExit
	if err := c.Bind(&param); err != nil {
		return err
	}
	du := time.Duration(param.Seconds)
	sec := du * time.Second
	if sec <= 0 {
		sec = time.Second
	}

	c.Warnf("[程序退出] %s 后程序将退出执行", sec)
	et.self.Exit(sec, 0)

	return nil
}

func (et *Exit) cancel(c *ship.Context) error {
	dead := et.self.Cancel()
	if !dead.IsZero() {
		c.Warnf("[程序退出] 程序将退出计划已取消")
	}

	return nil
}

func (et *Exit) deadline(c *ship.Context) error {
	dead := et.self.Deadline()
	ret := response.NewData(dead)

	return c.JSON(http.StatusOK, ret)
}
