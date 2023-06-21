package service

import (
	"context"

	"github.com/vela-ssoc/vela-common-mb/accord"

	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
)

type AgentService interface {
	UpgradeID(ctx context.Context, id int64) error
	Upgrade(ctx context.Context, goos, arch string) error
	Startup(ctx context.Context, id int64) error
	Command(ctx context.Context, id int64, cmd string) error
}

func Agent(lnk mlink.Huber, pool taskpool.Executor, slog logback.Logger) AgentService {
	return &agentService{
		lnk:  lnk,
		slog: slog,
		pool: pool,
	}
}

type agentService struct {
	lnk  mlink.Huber
	pool taskpool.Executor
	slog logback.Logger
}

func (biz *agentService) UpgradeID(ctx context.Context, id int64) error {
	path := "/api/v1/agent/notice/upgrade"
	return biz.lnk.Oneway(id, path, nil)
}

func (biz *agentService) Upgrade(ctx context.Context, goos, arch string) error {
	// TODO implement me
	panic("implement me")
}

func (biz *agentService) Startup(ctx context.Context, id int64) error {
	tsk := subtask.Startup(biz.lnk, id, biz.slog)
	biz.pool.Submit(tsk)
	return nil
}

func (biz *agentService) Command(ctx context.Context, id int64, cmd string) error {
	dat := &accord.Command{Cmd: cmd}
	path := "/api/v1/agent/notice/command"
	return biz.lnk.Oneway(id, path, dat)
}
