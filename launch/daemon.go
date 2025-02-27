package launch

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"

	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-common-mb/param/negotiate"
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
	srv := &http.Server{
		Addr:    srvCfg.Addr,
		Handler: ds.handler,
	}
	var enabledSSL bool
	if srvCfg.Cert != "" && srvCfg.Pkey == "" {
		cert, err := tls.X509KeyPair([]byte(srvCfg.Cert), []byte(srvCfg.Pkey))
		if err != nil {
			ds.errCh <- err
			return
		}
		enabledSSL = true
		srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	}

	if enabledSSL {
		ds.errCh <- srv.ListenAndServeTLS("", "")
	} else {
		ds.errCh <- srv.ListenAndServe()
	}
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
