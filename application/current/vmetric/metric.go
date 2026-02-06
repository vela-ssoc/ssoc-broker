package vmetric

import (
	"context"
	"io"
	"runtime"
	"strconv"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vela-ssoc/ssoc-common/datalayer/model"
)

type MetricWriter interface {
	WriteMetric(ctx context.Context, w io.Writer)
}

type ConfigLoader interface {
	LoadConfig(ctx context.Context) (pushURL string, opts *metrics.PushOptions, err error)
}

func Label(this *model.Broker) string {
	sid := strconv.FormatInt(this.ID, 10)
	return "instance=" + strconv.Quote(sid) +
		",instance_type=" + strconv.Quote("broker") +
		",instance_name=" + strconv.Quote(this.Name) +
		",goos=" + strconv.Quote(runtime.GOOS) +
		",goarch=" + strconv.Quote(runtime.GOOS)
}
