package service

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common/tundata/mbreq"
	"github.com/vela-ssoc/ssoc-common/tundata/mbresp"
	"github.com/vela-ssoc/ssoc-proto/muxconn"
	"golang.org/x/time/rate"
)

type Tunnel struct {
	mux brokclient.Muxer
	log *slog.Logger
}

func NewTunnel(mux brokclient.Muxer, log *slog.Logger) *Tunnel {
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

	stat := &mbresp.TunnelStat{
		Name:       name,
		Module:     module,
		Cumulative: cumulative,
		Active:     active,
		RX:         rx,
		TX:         tx,
		Limit:      float64(bps),
		Unlimit:    bps == rate.Inf,
	}

	streams := tnl.mux.Streams()
	for _, s := range streams {
		ss := s.Stats()
		stat.Streams = append(stat.Streams, ss)
	}
	slices.SortFunc(stat.Streams, func(a, b *muxconn.StreamStats) int {
		return cmp.Compare(a.ID, b.ID)
	})

	return stat
}

func (tnl *Tunnel) Limit(req *mbreq.TunnelLimit) {
	bps := req.Rate()
	tnl.mux.SetLimit(bps)
}

func (tnl *Tunnel) Kill(id uint64) {
	streams := tnl.mux.Streams()
	for _, stm := range streams {
		if stm.Stats().ID == id {
			stm.Close()
			break
		}
	}
}
