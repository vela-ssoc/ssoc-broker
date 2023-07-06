package service

import (
	"context"

	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/accord"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/logback"
)

type AgentService interface {
	UpgradeID(ctx context.Context, id int64, semver string) error
	Upgrade(ctx context.Context, goos, arch string) error
	Startup(ctx context.Context, id int64) error
	Offline(ctx context.Context, id int64) error
	Command(ctx context.Context, id int64, cmd string) error
}

func Agent(lnk mlink.Huber, pool gopool.Executor, slog logback.Logger) AgentService {
	return &agentService{
		lnk:  lnk,
		slog: slog,
		pool: pool,
	}
}

type agentService struct {
	lnk  mlink.Huber
	pool gopool.Executor
	slog logback.Logger
}

func (biz *agentService) UpgradeID(ctx context.Context, id int64, semver string) error {
	path := "/api/v1/agent/notice/upgrade"
	return biz.lnk.Oneway(id, path, &accord.Upgrade{Semver: semver})
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

func (biz *agentService) Offline(ctx context.Context, id int64) error {
	biz.lnk.Knockout(id)
	return nil
}

func (biz *agentService) Command(ctx context.Context, id int64, cmd string) error {
	dat := &accord.Command{Cmd: cmd}
	path := "/api/v1/agent/notice/command"
	return biz.lnk.Oneway(id, path, dat)
}

func (biz *agentService) Batch(ctx context.Context, id int64, cmd string) error {
	// dat := &accord.Command{Cmd: cmd}
	// path := "/api/v1/agent/notice/command"
	// return biz.lnk.Oneway(id, path, dat)
	return nil
}
