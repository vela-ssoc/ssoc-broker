package cronfunc

import (
	"context"
	"io"
	"time"

	"github.com/NVIDIA/gontainer/v2"
	"github.com/VictoriaMetrics/metrics"
	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-common-mb/crontab"
	"github.com/vela-ssoc/ssoc-common-mb/vmexport"
	"github.com/vela-ssoc/vela-tunnel/muxconn"
)

type VMetric struct {
	cfg    vmexport.VictoriaPushConfigLoader
	mux    muxconn.Muxer
	writes []vmexport.MetricWriter
}

func NewVMetric(cfg vmexport.VictoriaPushConfigLoader, mux muxconn.Muxer, writes gontainer.Multiple[vmexport.MetricWriter]) *VMetric {
	return &VMetric{cfg: cfg, mux: mux, writes: writes}
}

func (vm *VMetric) Options() crontab.JobOptions {
	return crontab.JobOptions{
		ID:      "86eb816810333dee1049300083b1e9a5b34f058a",
		Name:    "上报系统监控指标",
		Sched:   cron.Every(10 * time.Second),
		Timeout: 8 * time.Second,
	}
}

func (vm *VMetric) Execute(ctx context.Context) error {
	pushURL, opts, err := vm.cfg.LoadVictoriaPushConfig(ctx)
	if err != nil {
		return err
	}

	return metrics.PushMetricsExt(ctx, pushURL, vm.writeWithContext(ctx), opts)
}

func (vm *VMetric) writeWithContext(ctx context.Context) func(io.Writer) {
	return func(w io.Writer) {
		metrics.WritePrometheus(w, true)
		metrics.WriteFDMetrics(w)

		for _, mw := range vm.writes {
			mw.WriteMetric(ctx, w)
		}

		// 统计的是中继节点与中心端的通道指标数据
		rx, tx := vm.mux.Traffic()
		metrics.WriteCounterUint64(w, "tunnel_receive_bytes", rx)  // 中继节点虚拟通道接收字节数
		metrics.WriteCounterUint64(w, "tunnel_transmit_bytes", tx) // 中继节点虚拟通道发送字节数

		cumulative, active := vm.mux.NumStreams()
		metrics.WriteCounterUint64(w, "tunnel_cumulative_substreams", uint64(cumulative)) // 中继节点累计虚拟子流建立个数
		metrics.WriteCounterUint64(w, "tunnel_active_substreams", uint64(active))         // 中继节点当前活跃子流建立个数
	}
}
