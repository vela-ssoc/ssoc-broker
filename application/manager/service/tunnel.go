package service

import (
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/application/manager/request"
	"github.com/vela-ssoc/ssoc-broker/application/manager/response"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokcli"
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

func (tnl *Tunnel) Stat() *response.TunnelStat {
	rx, tx := tnl.mux.Traffic()
	bps := tnl.mux.Limit()

	return &response.TunnelStat{
		RX:        rx,
		TX:        tx,
		Limit:     float64(bps),
		Unlimited: bps == rate.Inf,
	}
}

func (tnl *Tunnel) Limit(req *request.TunnelLimit) {
	bps := req.Rate()
	tnl.mux.SetLimit(bps)
}
