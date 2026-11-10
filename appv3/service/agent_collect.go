package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type AgentCollect struct {
	pdb polydb.Sourcer
	log *slog.Logger
}

func NewAgentCollect(pdb polydb.Sourcer, log *slog.Logger) *AgentCollect {
	return &AgentCollect{pdb: pdb, log: log}
}

func (ac *AgentCollect) Sysinfo(ctx context.Context, node muxproto.Node, param request.MinionSysinfo) error {
	ac.log.Warn("TODO")
	return nil
}
