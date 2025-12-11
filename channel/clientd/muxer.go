package clientd

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/xtaci/smux"
)

type BootConfig struct {
	DSN         string        `json:"dsn"           validate:"required"`
	MaxOpenConn int           `json:"max_open_conn"`
	MaxIdleConn int           `json:"max_idle_conn"`
	MaxLifeTime time.Duration `json:"max_life_time"`
	MaxIdleTime time.Duration `json:"max_idle_time"`
}

type Muxer interface {
	// OpenConn 开启一个虚拟子流。
	OpenConn(ctx context.Context) (net.Conn, error)

	// BootConfig 连接中心端成功后，中心端给 broker 下发启动配置。
	BootConfig() BootConfig
}

type safeMuxer struct {
	mtx sync.RWMutex
	ses *smux.Session
	cfg BootConfig
}

func (sm *safeMuxer) OpenConn(context.Context) (net.Conn, error) {
	ses := sm.session()
	stm, err := ses.OpenStream()
	if err != nil {
		return nil, err
	}

	return stm, nil
}

func (sm *safeMuxer) BootConfig() BootConfig {
	sm.mtx.RLock()
	cfg := sm.cfg
	sm.mtx.RUnlock()

	return cfg
}

func (sm *safeMuxer) session() *smux.Session {
	sm.mtx.RLock()
	ses := sm.ses
	sm.mtx.RUnlock()

	return ses
}

func (sm *safeMuxer) replace(ses *smux.Session, cfg BootConfig) {
	sm.mtx.Lock()
	sm.ses = ses
	sm.cfg = cfg
	sm.mtx.Unlock()
}

type muxerListener struct {
	mux *safeMuxer
}

func (m *muxerListener) Accept() (net.Conn, error) {
	ses := m.mux.session()
	if stm, err := ses.AcceptStream(); err != nil {
		return nil, err
	} else {
		return stm, nil
	}
}

func (m *muxerListener) Close() error {
	ses := m.mux.session()
	return ses.Close()
}

func (m *muxerListener) Addr() net.Addr {
	ses := m.mux.session()
	return ses.LocalAddr()
}

func NewEqualDialer(mux Muxer, host string) *EqualDialer {
	return &EqualDialer{
		mux:  mux,
		host: host,
	}
}

type EqualDialer struct {
	host string
	mux  Muxer
}

func (e *EqualDialer) DialContext(ctx context.Context, _, addr string) (net.Conn, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	} else if host != e.host {
		return nil, nil
	}

	return e.mux.OpenConn(ctx)
}
