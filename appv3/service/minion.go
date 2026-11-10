package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type Minion struct {
	pdb polydb.Sourcer
	log *slog.Logger
}

func NewMinion(pdb polydb.Sourcer, log *slog.Logger) *Minion {
	return &Minion{pdb: pdb, log: log}
}

func (mon *Minion) Ping(ctx context.Context, node muxproto.Node) error {
	return nil
}
