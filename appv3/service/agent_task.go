package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/agentask"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
)

type AgentTask struct {
	cli agentask.Client
	pdb polydb.Sourcer
	log *slog.Logger
}

func NewAgentTask(cli agentask.Client, log *slog.Logger) *AgentTask {
	return &AgentTask{
		cli: cli,
		log: log,
	}
}

func (at *AgentTask) Resyncs(ctx context.Context, tags []string) error {
	return nil
}
