package agtaccept

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/vela-ssoc/vela-common-mba/smux"
	"golang.org/x/time/rate"
)

func newV1MUX(conn net.Conn, cfg *smux.Config) *velaSession {
	session := smux.Server(conn, cfg)

	return &velaSession{
		session: session,
		traffic: new(trafficStat),
		streams: new(streamStat),
	}
}

type velaSession struct {
	session *smux.Session
	traffic *trafficStat
	streams *streamStat
}

func (m *velaSession) Accept() (net.Conn, error) {
	return m.newConn(m.session.AcceptStream())
}

func (m *velaSession) Close() error {
	return m.session.Close()
}

func (m *velaSession) Addr() net.Addr {
	return m.session.LocalAddr()
}

func (m *velaSession) Open(context.Context) (net.Conn, error) {
	return m.newConn(m.session.OpenStream())
}

func (m *velaSession) RemoteAddr() net.Addr {
	return m.session.RemoteAddr()
}

func (m *velaSession) IsClosed() bool             { return m.session.IsClosed() }
func (m *velaSession) Limit() rate.Limit          { return rate.Inf }
func (m *velaSession) SetLimit(bps rate.Limit)    {}
func (m *velaSession) NumStreams() (int64, int64) { return m.streams.Load() }
func (m *velaSession) Traffic() (uint64, uint64)  { return m.traffic.Load() }

func (m *velaSession) Library() (string, string) {
	return "ssoc-smux", "github.com/vela-ssoc/vela-common-mba/smux"
}

func (m *velaSession) newConn(stm *smux.Stream, err error) (net.Conn, error) {
	if err != nil {
		return nil, err
	}

	m.streams.openOne()
	conn := &velaConn{
		parent: m,
		stream: stm,
	}

	return conn, nil
}

type velaConn struct {
	parent *velaSession
	stream *smux.Stream
	closed atomic.Bool
}

func (c *velaConn) Read(b []byte) (int, error) {
	n, err := c.stream.Read(b)
	c.parent.traffic.incrRX(n)

	return n, err
}

func (c *velaConn) Write(b []byte) (int, error) {
	n, err := c.stream.Write(b)
	c.parent.traffic.incrTX(n)

	return n, err
}

func (c *velaConn) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return net.ErrClosed
	}

	c.parent.streams.closeOne()
	err := c.stream.Close()

	return err
}

func (c *velaConn) LocalAddr() net.Addr                { return c.stream.LocalAddr() }
func (c *velaConn) RemoteAddr() net.Addr               { return c.stream.RemoteAddr() }
func (c *velaConn) SetDeadline(t time.Time) error      { return c.stream.SetDeadline(t) }
func (c *velaConn) SetReadDeadline(t time.Time) error  { return c.stream.SetReadDeadline(t) }
func (c *velaConn) SetWriteDeadline(t time.Time) error { return c.stream.SetWriteDeadline(t) }

type trafficStat struct {
	rx, tx atomic.Uint64
}

func (s *trafficStat) Load() (rx, tx uint64) {
	return s.rx.Load(), s.tx.Load()
}

func (s *trafficStat) incrRX(n int) {
	if n > 0 {
		s.rx.Add(uint64(n))
	}
}

func (s *trafficStat) incrTX(n int) {
	if n > 0 {
		s.tx.Add(uint64(n))
	}
}

type streamStat struct {
	cumulative, active atomic.Int64
}

func (s *streamStat) Load() (cumulative, active int64) {
	return s.cumulative.Load(), s.active.Load()
}

func (s *streamStat) openOne() {
	s.cumulative.Add(1)
	s.active.Add(1)
}

func (s *streamStat) closeOne() {
	s.active.Add(-1)
}
