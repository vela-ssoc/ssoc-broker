package vmetric

import (
	"context"
	"io"
	"net"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
	pscpu "github.com/shirou/gopsutil/v4/cpu"
	psmem "github.com/shirou/gopsutil/v4/mem"
	psnet "github.com/shirou/gopsutil/v4/net"
)

func NewPsutil() MetricWriter {
	return &psutilMetric{}
}

type psutilMetric struct{}

func (m *psutilMetric) WriteMetric(ctx context.Context, w io.Writer) {
	if stat, _ := psmem.VirtualMemoryWithContext(ctx); stat != nil {
		metrics.WriteCounterUint64(w, "system_memory_total_bytes", stat.Total)
		metrics.WriteCounterUint64(w, "system_memory_used_bytes", stat.Used)
	}

	if stats, _ := psnet.IOCountersWithContext(ctx, true); stats != nil {
		ignores := make(map[string]struct{}, 4)
		faces, _ := net.Interfaces()
		for _, face := range faces {
			if face.Flags&net.FlagLoopback != 0 { // 忽略环回网卡
				ignores[face.Name] = struct{}{}
			}
		}

		for _, stat := range stats {
			name := stat.Name
			if _, exists := ignores[name]; exists {
				continue
			}

			// 网卡名字转义
			name = strconv.Quote(name)
			metrics.WriteCounterUint64(w, "system_network_receive_bytes{device="+name+"}", stat.BytesRecv)
			metrics.WriteCounterUint64(w, "system_network_transmit_bytes{device="+name+"}", stat.BytesSent)
		}
	}

	if stats, _ := pscpu.PercentWithContext(ctx, 0, false); len(stats) != 0 {
		metrics.WriteGaugeFloat64(w, "system_cpu_usage_percent", stats[0])
	}
}
