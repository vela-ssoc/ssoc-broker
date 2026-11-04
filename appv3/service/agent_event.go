package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/muxtunnel/agtclient"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/gopool"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type AgentEvent struct {
	pdb polydb.Sourcer
	cli *agtclient.Client
	gop gopool.Pool
	log *slog.Logger
}

func NewAgentEvent(pdb polydb.Sourcer, cli *agtclient.Client, log *slog.Logger) *AgentEvent {
	return &AgentEvent{pdb: pdb, cli: cli, log: log}
}

func (ae *AgentEvent) OnConnected(node muxproto.Node) {
	ae.cli.PushStartup(context.Background(), node.Host(), model.StartupV1{})
}

func (ae *AgentEvent) OnDisconnected(node muxproto.Node) {
}
