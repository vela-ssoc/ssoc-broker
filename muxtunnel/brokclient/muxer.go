package brokclient

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	"github.com/vela-ssoc/vela-tunnel/muxconn"
	"golang.org/x/time/rate"
)

type Muxer interface {
	muxconn.Muxer
	Config() muxproto.BrokerBootConfig
}

type muxHolder struct {
	mux muxconn.Muxer
	cfg muxproto.BrokerBootConfig
}

type safeMUX struct {
	ptr atomic.Pointer[muxHolder] // 当返回给调用者时，一定有数据。
}

func (s *safeMUX) Open(ctx context.Context) (net.Conn, error) { return s.loadMUX().Open(ctx) }
func (s *safeMUX) Accept() (net.Conn, error)                  { return s.loadMUX().Accept() }
func (s *safeMUX) Close() error                               { return s.loadMUX().Close() }
func (s *safeMUX) Addr() net.Addr                             { return s.loadMUX().Addr() }
func (s *safeMUX) RemoteAddr() net.Addr                       { return s.loadMUX().RemoteAddr() }
func (s *safeMUX) IsClosed() bool                             { return s.loadMUX().IsClosed() }
func (s *safeMUX) Limit() rate.Limit                          { return s.loadMUX().Limit() }
func (s *safeMUX) SetLimit(bps rate.Limit) bool               { return s.loadMUX().SetLimit(bps) }
func (s *safeMUX) NumStreams() (int64, int64)                 { return s.loadMUX().NumStreams() }
func (s *safeMUX) Traffic() (uint64, uint64)                  { return s.loadMUX().Traffic() }
func (s *safeMUX) Library() (string, string)                  { return s.loadMUX().Library() }
func (s *safeMUX) Streams() []muxconn.Streamer                { return s.loadMUX().Streams() }
func (s *safeMUX) ConnectedAt() time.Time                     { return s.loadMUX().ConnectedAt() }
func (s *safeMUX) Config() muxproto.BrokerBootConfig          { return s.ptr.Load().cfg }

func (s *safeMUX) loadMUX() muxconn.Muxer {
	return s.ptr.Load().mux
}

func (s *safeMUX) store(mux muxconn.Muxer, pcfg *muxproto.BrokerBootConfig) {
	var cfg muxproto.BrokerBootConfig
	if pcfg != nil {
		cfg = *pcfg
	}

	hold := &muxHolder{mux: mux, cfg: cfg}
	s.ptr.Store(hold)
}
