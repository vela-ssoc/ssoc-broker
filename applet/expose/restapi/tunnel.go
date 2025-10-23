package restapi

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/ssoc-broker/channel/serverd"
	"github.com/xgfone/ship/v5"
	"github.com/xtaci/smux"
)

func NewTunnel(next serverd.Handler) *Tunnel {
	return &Tunnel{
		next: next,
		wsup: &websocket.Upgrader{
			HandshakeTimeout:  30 * time.Second,
			CheckOrigin:       func(*http.Request) bool { return true },
			EnableCompression: false,
		},
	}
}

type Tunnel struct {
	next serverd.Handler
	wsup *websocket.Upgrader
}

func (tnl *Tunnel) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/tunnel").GET(tnl.open)

	return nil
}

func (tnl *Tunnel) open(c *ship.Context) error {
	w, r := c.Response(), c.Request()
	ws, err := tnl.wsup.Upgrade(w, r, nil)
	if err != nil {
		c.Warnf("agent websocket 隧道升级失败", "error", err)
		return nil
	}

	conn := ws.NetConn()
	cfg := smux.DefaultConfig()
	cfg.KeepAliveDisabled = true
	sess, err1 := smux.Server(conn, cfg)
	if err1 != nil {
		c.Errorf("转为 smux 多路复用失败", "error", err)
		_ = conn.Close()
		return nil
	}

	tnl.next.Handle(sess)

	return nil
}
