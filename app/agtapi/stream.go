package agtapi

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/internal/stmsrv"
	"github.com/vela-ssoc/vela-broker/app/middle"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/integration/elastic"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/xgfone/ship/v5"
)

func Stream(name string, esc elastic.Searcher) route.Router {
	schemes := make(map[string]stmsrv.Upstreamer, 16)
	nt := stmsrv.Net()
	htp := stmsrv.HTTP()
	for _, s := range nt.Proto() {
		schemes[s] = nt
	}
	for _, s := range htp.Proto() {
		schemes[s] = htp
	}

	rest := &streamREST{name: name, esc: esc, schemes: schemes}
	rest.upgrade = netutil.Upgrade(rest.upgradeError)

	return rest
}

type streamREST struct {
	name    string
	upgrade websocket.Upgrader
	esc     elastic.Searcher
	schemes map[string]stmsrv.Upstreamer
}

func (rest *streamREST) Route(r *ship.RouteGroupBuilder) {
	rt := r.Group("/broker/stream", middle.MustWebsocket)
	rt.Route("/tunnel").GET(rest.Tunnel)
	rt.Route("/elastic").GET(rest.Elastic)
}

func (rest *streamREST) Tunnel(c *ship.Context) error {
	var req param.TunnelRequest
	if err := c.BindQuery(&req); err != nil {
		return err
	}

	w, r := c.Response(), c.Request()
	info := mlink.Ctx(r.Context())
	inet := info.Inet()
	mid := info.Issue().ID
	addr := req.Address
	c.Infof("节点 %s(%d) 准备建立通道代理：%s", inet, mid, addr)

	u, err := url.Parse(addr)
	if err != nil {
		c.Warnf("agent 传来的 addr 格式错误：%s", err)
		return err
	}
	scheme := u.Scheme
	proto, ok := rest.schemes[scheme]
	if !ok {
		c.Warnf("不支持的协议：%s", scheme)
		return ship.ErrBadRequest.Newf(scheme)
	}

	conn, err := proto.Dial(u, req.Skip)
	if err != nil {
		c.Warnf("proto dial 错误：%s", err)
		return err
	}

	ws, err := rest.upgrade.Upgrade(w, r, nil)
	if err != nil {
		c.Warnf("节点 %s(%d) 准备建立通道代理：%s", inet, mid, addr)
		_ = conn.Close()
		return err
	}

	if err = proto.Serve(u, conn, ws); err != nil && err != context.Canceled {
		c.Infof("代理错误：%s", err)
	}
	c.Infof("节点 %s(%d) 代理通道 %s 关闭", inet, mid, addr)

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
