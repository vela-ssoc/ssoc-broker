package restapi

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/ssoc-common/linkhub"
	"github.com/xgfone/ship/v5"
	"github.com/xtaci/smux"
)

func NewTunnel(next linkhub.Handler) *Tunnel {
	return &Tunnel{
		next: next,
		wsup: &websocket.Upgrader{
			HandshakeTimeout: 10 * time.Second,
			CheckOrigin:      func(r *http.Request) bool { return true },
		},
	}
}

type Tunnel struct {
	next linkhub.Handler
	wsup *websocket.Upgrader
}

func (tnl *Tunnel) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/tunnel").GET(tnl.open)

	return nil
}

// open agent 的接入点。
func (tnl *Tunnel) open(c *ship.Context) error {
	w, r := c.Response(), c.Request()
	ws, err := tnl.wsup.Upgrade(w, r, nil)
	if err != nil {
		c.Warnf("websocket 协议升级错误", "error", err)
		return nil
	}
	defer ws.Close()
	conn := ws.NetConn()
	sess, err := smux.Server(conn, nil)
	if err != nil {
		_ = conn.Close()
		c.Warnf("升级为 smux 协议错误", "error", err)
		return nil
	}
	tnl.next.Handle(sess)

	return nil
}
