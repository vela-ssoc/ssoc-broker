package agtsvc

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/app/internal/param"
	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"gorm.io/gorm/clause"
)

func NewAgentEmergencySnapshot(qry *query.Query, log *slog.Logger) *AgentEmergencySnapshot {
	return &AgentEmergencySnapshot{
		qry: qry,
		log: log,
	}
}

type AgentEmergencySnapshot struct {
	qry *query.Query
	log *slog.Logger
}

func (aes *AgentEmergencySnapshot) Report(ctx context.Context, req *param.AgentEmergencySnapshotReport, info mlink.Infer) error {
	minionID := info.Issue().ID
	inet := info.Inet().String()

	tbl := aes.qry.AgentEmergencySnapshot
	dao := tbl.WithContext(ctx)

	dat := &model.AgentEmergencySnapshot{
		MinionID:   minionID,
		Inet:       inet,
		Type:       req.Type,
		Value:      req.Value,
		ReportedAt: req.ReportedAt,
	}
	err := dao.Where(tbl.MinionID.Eq(minionID), tbl.Inet.Eq(inet)).
		Clauses(clause.OnConflict{UpdateAll: true}).
		Create(dat)

	return err
}
