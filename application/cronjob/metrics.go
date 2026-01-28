package cronjob

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"github.com/vela-ssoc/ssoc-common/datalayer/model"
)

type MetricsConfigFunc func(ctx context.Context) (pushURL string, opts *metrics.PushOptions, err error)

func NewMetrics(this *model.Broker, cfg MetricsConfigFunc) cronv3.Tasker {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	label := fmt.Sprintf(`instance="%d",instance_type="broker",instance_name="%s",goos="%s",goarch="%s"`, this.ID, this.Name, goos, goarch)

	return &metricsJob{
		cfg:   cfg,
		label: label,
	}
}

type metricsJob struct {
	cfg   MetricsConfigFunc
	label string
}

func (vm *metricsJob) Info() cronv3.TaskInfo {
	return cronv3.TaskInfo{
		Name:      "上报系统指标",
		Timeout:   9 * time.Second,
		CronSched: cron.Every(10 * time.Second),
	}
}

func (vm *metricsJob) Call(ctx context.Context) error {
	pushURL, opts, err := vm.cfg(ctx)
	if err != nil {
		return err
	}
	opts.ExtraLabels = vm.label

	return metrics.PushMetricsExt(ctx, pushURL, vm.defaultWrite, opts)
}

func (*metricsJob) defaultWrite(w io.Writer) {
	metrics.WritePrometheus(w, true)
	metrics.WriteFDMetrics(w)
}
