package cronjob

import (
	"context"
	"io"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"github.com/vela-ssoc/ssoc-common/vmetric"
)

func NewMetrics(label string, cfg vmetric.ConfigLoader, writers []vmetric.MetricWriter) cronv3.Tasker {
	return &metricsJob{
		label:   label,
		cfg:     cfg,
		writers: writers,
	}
}

type metricsJob struct {
	label   string
	cfg     vmetric.ConfigLoader
	writers []vmetric.MetricWriter
}

func (*metricsJob) Info() cronv3.TaskInfo {
	return cronv3.TaskInfo{
		Name:      "上报系统指标",
		Timeout:   9 * time.Second,
		CronSched: cron.Every(10 * time.Second),
	}
}

func (m *metricsJob) Call(ctx context.Context) error {
	pushURL, opts, err := m.cfg.LoadConfig(ctx)
	if err != nil {
		return err
	}
	opts.ExtraLabels = m.label

	return metrics.PushMetricsExt(ctx, pushURL, m.writeWithContext(ctx), opts)
}

func (m *metricsJob) writeWithContext(ctx context.Context) func(io.Writer) {
	return func(w io.Writer) {
		metrics.WritePrometheus(w, true)
		metrics.WriteFDMetrics(w)

		for _, mw := range m.writers {
			mw.WriteMetric(ctx, w)
		}
	}
}
