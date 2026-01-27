package brokcli

import (
	"context"
	"net"
	"sync/atomic"

	"github.com/vela-ssoc/ssoc-proto/muxconn"
	"golang.org/x/time/rate"
)

type Muxer interface {
	muxconn.Muxer
	Config() BrokConfig
}

type muxHolder struct {
	mux muxconn.Muxer
	cfg BrokConfig
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
func (s *safeMUX) SetLimit(bps rate.Limit)                    { s.loadMUX().SetLimit(bps) }
func (s *safeMUX) NumStreams() (int64, int64)                 { return s.loadMUX().NumStreams() }
func (s *safeMUX) Traffic() (uint64, uint64)                  { return s.loadMUX().Traffic() }
func (s *safeMUX) Library() (string, string)                  { return s.loadMUX().Library() }
func (s *safeMUX) Config() BrokConfig                         { return s.ptr.Load().cfg }

func (s *safeMUX) loadMUX() muxconn.Muxer {
	return s.ptr.Load().mux
}

func (s *safeMUX) store(mux muxconn.Muxer, cfg BrokConfig) {
	hold := &muxHolder{mux: mux, cfg: cfg}
	s.ptr.Store(hold)
}
