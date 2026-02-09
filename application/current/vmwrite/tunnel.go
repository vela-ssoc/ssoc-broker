package vmwrite

import (
	"context"
	"io"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokcli"
	"github.com/vela-ssoc/ssoc-common/vmetric"
)

func NewTunnel(mux brokcli.Muxer) vmetric.MetricWriter {
	return &tunnelMetric{
		mux: mux,
	}
}

type tunnelMetric struct {
	mux brokcli.Muxer
}

func (m *tunnelMetric) WriteMetric(_ context.Context, w io.Writer) {
	rx, tx := m.mux.Traffic()
	metrics.WriteCounterUint64(w, "tunnel_receive_bytes", rx)
	metrics.WriteCounterUint64(w, "tunnel_transmit_bytes", tx)

	cumulative, active := m.mux.NumStreams()
	metrics.WriteCounterUint64(w, "tunnel_cumulative_streams", uint64(cumulative))
	metrics.WriteCounterUint64(w, "tunnel_active_streams", uint64(active))
}
