package launch

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net"
	"net/http"

	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-common-mb/param/negotiate"
	"github.com/vela-ssoc/vela-common-mb/prereadtls"
)

type daemonServer struct {
	hide    *negotiate.Hide // 隐写配置
	issue   negotiate.Issue // 服务监听配置
	handler http.Handler    // handler
	server  *http.Server    // HTTP 服务
	errCh   chan<- error    // 错误输出
}

func (ds *daemonServer) Run() {
	srvCfg := ds.issue.Server
	addr := srvCfg.Addr
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		ds.errCh <- err
		return
	}
	//goland:noinspection GoUnhandledErrorResult
	defer lis.Close()

	tcpSrv := &http.Server{Handler: ds.handler}

	var tlsFunc func(net.Conn)
	cert, pkey := srvCfg.Cert, srvCfg.Pkey
	if cert != "" && pkey == "" {
		pair, err := tls.X509KeyPair([]byte(cert), []byte(pkey))
		if err != nil {
			ds.errCh <- err
			return
		}

		tcpSrv.Handler = &onlyDeploy{h: tcpSrv.Handler}
		tlsSrv := &http.Server{
			Handler:   ds.handler,
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		}

		tlsFunc = func(conn net.Conn) {
			ln := prereadtls.NewOnceAccept(conn)
			_ = tlsSrv.ServeTLS(ln, "", "")
		}
	}
	tcpFunc := func(conn net.Conn) {
		ln := prereadtls.NewOnceAccept(conn)
		_ = tcpSrv.Serve(ln)
	}

	ds.errCh <- prereadtls.Serve(lis, tcpFunc, tlsFunc)
}

func (ds *daemonServer) Close() error {
	if srv := ds.server; srv != nil {
		return srv.Close()
	}
	return nil
}

type daemonClient struct {
	link    telecom.Linker
	handler http.Handler
	server  *http.Server
	errCh   chan<- error
	log     *slog.Logger
	parent  context.Context
}

func (dc *daemonClient) Run() {
	for {
		lis := dc.link.Listen()
		dc.server = &http.Server{Handler: dc.handler}
		_ = dc.server.Serve(lis)
		dc.log.Warn("与中心端的连接已断开")
		if err := dc.parent.Err(); err != nil {
			dc.errCh <- err
			break
		}
		dc.log.Info("正在准备重试连接中心端")
		if err := dc.link.Reconnect(dc.parent); err != nil {
			dc.errCh <- err
			break
		}
		dc.log.Info("重新连接中心端成功")
	}
}

func (dc *daemonClient) Close() error {
	if srv := dc.server; srv != nil {
		return srv.Close()
	}
	return nil
}

type onlyDeploy struct {
	h http.Handler
}

func (od *onlyDeploy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	allows := map[string]struct{}{
		"/api/v1/deploy/minion":           {},
		"/api/v1/deploy/minion/":          {},
		"/api/v1/deploy/minion/download":  {},
		"/api/v1/deploy/minion/download/": {},
	}
	path := r.URL.Path
	if _, allow := allows[path]; allow {
		od.h.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusUpgradeRequired)
	}
}
