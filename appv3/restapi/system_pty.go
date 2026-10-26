package restapi

import (
	"net"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/pterm"
	"github.com/xgfone/ship/v5"
)

type SystemPTY struct {
	wsu *websocket.Upgrader
}

func NewSystemPTY(wsu *websocket.Upgrader) *SystemPTY {
	return &SystemPTY{wsu: wsu}
}

func (pt *SystemPTY) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/system/pty").GET(pt.open)

	return nil
}

//goland:noinspection ALL
func (pt *SystemPTY) open(c *ship.Context) error {
	w, r := c.Response(), c.Request()
	ws, err := pt.wsu.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	err = pterm.HandlePTY(ws, "")
	if err == nil {
		return nil
	} else if _, ok := err.(*websocket.CloseError); ok {
		return nil
	} else if _, yes := err.(*net.OpError); yes {
		return nil
	}
	c.Errorf("PTY 运行出错: %s", err.Error())

	return nil
}
