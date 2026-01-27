package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/vela-ssoc/ssoc-common/datalayer/model"
	"github.com/vela-ssoc/ssoc-common/datalayer/query"
)

type Broker struct {
	secret string
	qry    *query.Query
	log    *slog.Logger
}

func NewBroker(secret string, qry *query.Query, log *slog.Logger) *Broker {
	return &Broker{
		secret: secret,
		qry:    qry,
		log:    log,
	}
}

func (brk *Broker) Get(ctx context.Context) (*model.Broker, error) {
	tbl := brk.qry.Broker
	dao := tbl.WithContext(ctx)

	return dao.Where(tbl.Secret.Eq(brk.secret)).First()
}

// ResetAgents 将当前 broker 节点下的所有 agent 标记为下线。
func (brk *Broker) ResetAgents(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	this, err := brk.Get(ctx)
	if err != nil {
		return err
	}

	tbl := brk.qry.Minion
	dao := tbl.WithContext(ctx)

	online, offline := uint8(model.MSOnline), uint8(model.MSOffline)
	_, err = dao.Where(tbl.Status.Value(online), tbl.BrokerID.Eq(this.ID)).
		UpdateColumnSimple(tbl.Status.Value(offline))

	return err
}
