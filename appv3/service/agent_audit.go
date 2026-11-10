package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type AgentAudit struct {
	pdb polydb.Sourcer
	log *slog.Logger
}

func NewAgentAudit(pdb polydb.Sourcer, log *slog.Logger) *AgentAudit {
	return &AgentAudit{pdb: pdb, log: log}
}

func (aa *AgentAudit) Event(ctx context.Context, node muxproto.Node, param request.AgentAuditEvent) error {
	aa.log.Warn("TODO")
	return nil
}

func (aa *AgentAudit) Risk(ctx context.Context, node muxproto.Node, param request.AgentAuditRisk) error {
	aa.log.Warn("TODO")
	return nil
}
