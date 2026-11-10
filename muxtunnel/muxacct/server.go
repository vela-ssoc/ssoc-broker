package muxacct

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type Eventer interface {
	// OnConnected 上线事件（不要阻塞调用）。
	OnConnected(node muxproto.Node)

	// OnDisconnected 下线事件（不要阻塞调用）。
	OnDisconnected(node muxproto.Node)
}

type Options struct {
	Event      Eventer
	Validate   func(any) error
	PerTimeout time.Duration
}

type Server struct {
	pdb  polydb.Sourcer
	mux  brokclient.Muxer
	hub  muxproto.Huber
	next http.Handler
	log  *slog.Logger
	opts Options
}

func NewServer(pdb polydb.Sourcer, mux brokclient.Muxer, hub muxproto.Huber, next http.Handler, log *slog.Logger, opts Options) *Server {
	evt := &safeEvent{event: opts.Event}
	opts.Event = evt

	return &Server{
		pdb:  pdb,
		mux:  mux,
		hub:  hub,
		next: next,
		log:  log,
		opts: opts,
	}
}

type safeEvent struct {
	event Eventer
}

func (se *safeEvent) OnConnected(node muxproto.Node) {
	if se.event != nil {
		se.event.OnConnected(node)
	}
}

func (se *safeEvent) OnDisconnected(node muxproto.Node) {
	if se.event != nil {
		se.event.OnDisconnected(node)
	}
}
