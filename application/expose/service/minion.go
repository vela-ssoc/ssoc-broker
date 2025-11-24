package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
)

func NewMinion(qry *query.Query, log *slog.Logger) *Minion {
	return &Minion{
		qry: qry,
		log: log,
	}
}

type Minion struct {
	qry *query.Query
	log *slog.Logger
}

func (mn *Minion) Reset(ctx context.Context, brokerID int64) error {
	online, offline := uint8(model.MSOnline), uint8(model.MSOffline)
	tbl := mn.qry.Minion
	dao := tbl.WithContext(ctx)
	_, err := dao.Where(tbl.BrokerID.Eq(brokerID), tbl.Status.Eq(online)).
		UpdateSimple(tbl.Status.Value(offline))

	return err
}
