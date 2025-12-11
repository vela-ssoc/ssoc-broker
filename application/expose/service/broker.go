package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
)

func NewBroker(qry *query.Query, log *slog.Logger) *Broker {
	return &Broker{
		qry: qry,
		log: log,
	}
}

type Broker struct {
	qry *query.Query
	log *slog.Logger
}

func (brok *Broker) Load(ctx context.Context, secret string) (*model.Broker, error) {
	tbl := brok.qry.Broker
	dao := tbl.WithContext(ctx)

	return dao.Where(tbl.Secret.Eq(secret)).First()
}
