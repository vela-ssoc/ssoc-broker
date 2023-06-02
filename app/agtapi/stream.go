package agtapi

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/middle"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/xgfone/ship/v5"
)

func Stream(name string, esc elastic.Searcher) route.Router {
	rest := &streamREST{name: name, esc: esc}
	rest.upgrade = netutil.Upgrade(rest.upgradeError)

	return rest
}

type streamREST struct {
	name    string
	upgrade websocket.Upgrader
	esc     elastic.Searcher
}

func (rest *streamREST) Route(r *ship.RouteGroupBuilder) {
	rt := r.Group("/broker/stream", middle.MustWebsocket)
	rt.Route("/tunnel").GET(rest.Tunnel)
	rt.Route("/elastic").GET(rest.Elastic)
}

func (rest *streamREST) Tunnel(c *ship.Context) error {
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

func (rest *streamREST) Elastic(c *ship.Context) error {
	w, r := c.Response(), c.Request()
	parent := r.Context()
	inf := mlink.Ctx(parent)
	inet, id := inf.Inet(), inf.Issue().ID

	ws, err := rest.upgrade.Upgrade(w, r, nil)
	if err != nil {
		c.Warnf("节点 %s(%d) es tunnel upgrade 失败：%s", err, inet, id)
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer ws.Close()

	c.Infof("节点 %s(%d) 建立 es 代理成功", inet, id)

	var n int
	for {
		//_, dat, err := ws.ReadMessage()
		_, rd, err := ws.NextReader()
		if err != nil {
			c.Warnf("es tunnel next reader 获取失败：%s", err)
			break
		}
		n++
		ctx, cancel := context.WithTimeout(parent, 3*time.Second)
		res, err := rest.esc.Bulk(ctx, rd)
		cancel()
		log.Printf("es %d : %v", n, err)
		if err != nil {
			c.Warnf("es bulk 写入错误：%s", err)
			continue
		}
		if res.Errors {
			c.Warnf("es bulk 写入存在错误数据")
		}
	}
	c.Infof("节点 %s(%d) 关闭 es 代理", inet, id)

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
