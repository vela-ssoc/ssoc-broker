package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Broker struct {
	db  repository.Database
	id  bson.ObjectID
	log *slog.Logger
}

func NewBroker(db repository.Database, id bson.ObjectID, log *slog.Logger) *Broker {
	return &Broker{
		db:  db,
		id:  id,
		log: log,
	}
}

// ResetAgents 将当前 broker 节点下的所有 agent 标记为下线。
func (brk *Broker) ResetAgents(timeout time.Duration) error {
	filter := bson.M{"broker.id": brk.id, "status": model.MinionStatusOnline}
	update := bson.M{"$set": bson.M{"status": model.MinionStatusOffline}}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	coll := brk.db.Minion()
	_, err := coll.UpdateMany(ctx, filter, update)

	return err
}
