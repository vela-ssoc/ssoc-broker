package launch

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-common-mb/logback"
)

type daemonServer struct {
	listen  telecom.Listen // 服务监听配置
	handler http.Handler   // handler
	server  *http.Server   // HTTP 服务
	errCh   chan<- error   // 错误输出
}

func (ds *daemonServer) Run() {
	certs, err := ds.listen.Certifier()
	if err != nil {
		ds.errCh <- err
		return
	}

	srv := &http.Server{
		Addr:    ds.listen.Addr,
		Handler: ds.handler,
	}
	if len(certs) != 0 {
		srv.TLSConfig = &tls.Config{Certificates: certs}
		err = srv.ListenAndServeTLS("", "")
	} else {
		err = srv.ListenAndServe()
	}

	ds.errCh <- err
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
	slog    logback.Logger
	parent  context.Context
}

func (dc *daemonClient) Run() {
	for {
		lis := dc.link.Listen()
		dc.server = &http.Server{Handler: dc.handler}
		_ = dc.server.Serve(lis)
		dc.slog.Warn("与中心端的连接已断开")
		if err := dc.parent.Err(); err != nil {
			dc.errCh <- err
			break
		}
		dc.slog.Info("正在准备重试连接中心端")
		if err := dc.link.Reconnect(dc.parent); err != nil {
			dc.errCh <- err
			break
		}
		dc.slog.Info("重新连接中心端成功")
	}
}

func (dc *daemonClient) Close() error {
	if srv := dc.server; srv != nil {
		return srv.Close()
	}
	return nil
}
