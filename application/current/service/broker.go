package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
)

type Broker struct {
	db     repository.Database
	secret string
	log    *slog.Logger
}

func NewBroker(db repository.Database, secret string, log *slog.Logger) *Broker {
	return &Broker{
		db:     db,
		secret: secret,
		log:    log,
	}
}

func (brk *Broker) Get(ctx context.Context) (*model.Broker, error) {
	coll := brk.db.Broker()

	return coll.FindBySecret(ctx, brk.secret)
}

// ResetAgents 将当前 broker 节点下的所有 agent 标记为下线。
func (brk *Broker) ResetAgents(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	_, err := brk.Get(ctx)
	if err != nil {
		return err
	}

	//coll := brk.db.Broker()
	//
	//online, offline := uint8(model.MSOnline), uint8(model.MSOffline)
	//_, err = dao.Where(tbl.Status.Value(online), tbl.BrokerID.Eq(this.ID)).
	//	UpdateColumnSimple(tbl.Status.Value(offline))

	return err
}
