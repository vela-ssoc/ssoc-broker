package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/vela-ssoc/ssoc-broker/agentask"
	"github.com/vela-ssoc/ssoc-common-mb/gopool"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type AgentConnectEvent struct {
	cli  agentask.Client
	pool gopool.Pool
	log  *slog.Logger
}

func NewAgentConnectEvent(cli agentask.Client, pool gopool.Pool, log *slog.Logger) *AgentConnectEvent {
	return &AgentConnectEvent{cli: cli, pool: pool, log: log}
}

func (ace *AgentConnectEvent) OnConnected(node muxproto.Node) {
	fn := ace.onConnected(node)
	ace.pool.Go(fn)
}

func (ace *AgentConnectEvent) OnDisconnected(node muxproto.Node) {
}

func (ace *AgentConnectEvent) onConnected(node muxproto.Node) func() {
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()

		idv1 := node.Info().IDV1
		_ = ace.cli.PushStartup(ctx, idv1)
		_ = ace.cli.ResyncTasks(ctx, idv1, 3)
	}
}
