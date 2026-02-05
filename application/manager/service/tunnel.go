package service

import (
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokcli"
	"github.com/vela-ssoc/ssoc-common/tundata/mbreq"
	"github.com/vela-ssoc/ssoc-common/tundata/mbresp"
	"golang.org/x/time/rate"
)

type Tunnel struct {
	mux brokcli.Muxer
	log *slog.Logger
}

func NewTunnel(mux brokcli.Muxer, log *slog.Logger) *Tunnel {
	return &Tunnel{
		mux: mux,
		log: log,
	}
}

func (tnl *Tunnel) Stat() *mbresp.TunnelStat {
	cumulative, active := tnl.mux.NumStreams()
	name, module := tnl.mux.Library()
	rx, tx := tnl.mux.Traffic()
	bps := tnl.mux.Limit()

	return &mbresp.TunnelStat{
		Name:       name,
		Module:     module,
		Cumulative: cumulative,
		Active:     active,
		RX:         rx,
		TX:         tx,
		Limit:      float64(bps),
		Unlimit:    bps == rate.Inf,
	}
}

func (tnl *Tunnel) Limit(req *mbreq.TunnelLimit) {
	bps := req.Rate()
	tnl.mux.SetLimit(bps)
}
