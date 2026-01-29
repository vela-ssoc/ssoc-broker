package cronjob

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokcli"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"github.com/vela-ssoc/ssoc-common/datalayer/model"
)

type MetricsConfigFunc func(ctx context.Context) (pushURL string, opts *metrics.PushOptions, err error)

func NewMetrics(this *model.Broker, mux brokcli.Muxer, cfg MetricsConfigFunc) cronv3.Tasker {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	label := fmt.Sprintf(`instance="%d",instance_type="broker",instance_name="%s",goos="%s",goarch="%s"`, this.ID, this.Name, goos, goarch)

	return &metricsJob{
		mux:   mux,
		cfg:   cfg,
		label: label,
	}
}

type metricsJob struct {
	mux   brokcli.Muxer
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

func (vm *metricsJob) defaultWrite(w io.Writer) {
	metrics.WritePrometheus(w, true)
	metrics.WriteFDMetrics(w)

	rx, tx := vm.mux.Traffic()
	rxName := fmt.Sprintf("tunnel_receive_bytes{%s}", vm.label)
	txName := fmt.Sprintf("tunnel_transmit_bytes{%s}", vm.label)
	metrics.WriteCounterUint64(w, rxName, rx)
	metrics.WriteCounterUint64(w, txName, tx)
}
