package cronjob

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/mgtclient"
	"github.com/vela-ssoc/ssoc-common/cronv3"
)

func NewHeartbeat(cli mgtclient.Client, log *slog.Logger) cronv3.Tasker {
	return &heartbeatPacket{
		cli: cli,
		log: log,
	}
}

type heartbeatPacket struct {
	cli   mgtclient.Client
	log   *slog.Logger
	fails int // 连续失败的次数
}

func (h *heartbeatPacket) Info() cronv3.TaskInfo {
	return cronv3.TaskInfo{
		Name:      "发送心跳包",
		Timeout:   10 * time.Second,
		CronSched: cron.Every(time.Minute),
	}
}

func (h *heartbeatPacket) Call(ctx context.Context) error {
	if err := h.cli.Ping(ctx); err != nil {
		h.fails++
		if h.fails < 3 {
			h.log.Warn("心跳包发送错误", "error", err, "fails", h.fails)
		} else {
			h.log.Error("连续多次心跳包发送错误", "error", err, "fails", h.fails)
		}

		return err
	}

	h.fails = 0

	return nil
}
