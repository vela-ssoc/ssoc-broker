package agtapi

import (
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/xgfone/ship/v5"
)

func Stream(name string) route.Router {
	rest := &streamREST{name: name}
	rest.upgrade = netutil.Upgrade(rest.upgradeError)

	return rest
}

type streamREST struct {
	name    string
	upgrade websocket.Upgrader
}

func (rest *streamREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/stream/tunnel").GET(rest.Tunnel)
}

func (rest *streamREST) Tunnel(c *ship.Context) error {
	if !c.IsWebSocket() {
		return ship.ErrBadRequest
	}
	var req param.TunnelTCP
	if err := c.BindQuery(&req); err != nil {
		return err
	}

	addr := req.Address
	w, r := c.Response(), c.Request()
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer conn.Close()

	info := mlink.Ctx(r.Context())
	inet := info.Inet()
	mid := info.Issue().ID

	ws, err := rest.upgrade.Upgrade(w, r, nil)
	if err != nil {
		_ = conn.Close()
		return err
	}

	maddr, laddr, raddr := r.RemoteAddr, conn.LocalAddr(), conn.RemoteAddr()
	c.Infof("为节点 %s(%d) 建立 TCP 隧道 %s(ws) <=> %s <=> %s(%s)", inet, mid, maddr, laddr, addr, raddr)
	netutil.ConnSockPIPE(conn, ws)
	c.Infof("节点 %s(%d) 的 TCP 隧道关闭 %s(ws) <=> %s <=> %s(%s)", inet, mid, maddr, laddr, addr, raddr)

	return nil
}

func (rest *streamREST) upgradeError(w http.ResponseWriter, r *http.Request, code int, err error) {
	pd := &problem.Detail{
		Type:     rest.name,
		Title:    "stream 模块升级 websocket 错误",
		Status:   code,
		Detail:   err.Error(),
		Instance: r.RequestURI,
	}
	_ = pd.JSON(w)
}
