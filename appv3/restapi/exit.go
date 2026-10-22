package restapi

import (
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/response"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/xgfone/ship/v5"
)

type Exit struct {
	mutex sync.Mutex
	timer *time.Timer
	dead  time.Time
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

	et.mutex.Lock()
	defer et.mutex.Unlock()

	if et.timer != nil {
		et.timer.Stop()
	}

	now := time.Now()
	et.dead = now.Add(sec)
	et.timer = time.AfterFunc(sec, func() {
		os.Exit(0)
	})

	return nil
}

func (et *Exit) cancel(c *ship.Context) error {
	var canceled bool
	et.mutex.Lock()
	if et.timer != nil {
		et.timer.Stop()
		et.timer = nil
		et.dead = time.Time{}
		canceled = true
	}
	et.mutex.Unlock()

	if canceled {
		c.Warnf("[程序退出] 程序将退出计划已取消")
	}

	return nil
}

func (et *Exit) deadline(c *ship.Context) error {
	et.mutex.Lock()
	timer, dead := et.timer, et.dead
	et.mutex.Unlock()

	var ret response.Data[time.Time]
	if timer != nil {
		ret.Data = dead
	}

	return c.JSON(http.StatusOK, ret)
}
