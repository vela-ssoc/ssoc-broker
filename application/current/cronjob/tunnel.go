package cronjob

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func NewTunnelStat(db repository.Database, id bson.ObjectID, mux brokclient.Muxer) cronv3.Tasker {
	return &tunnelStat{
		db:  db,
		id:  id,
		mux: mux,
	}
}

type tunnelStat struct {
	db  repository.Database
	id  bson.ObjectID
	mux brokclient.Muxer
}

func (t *tunnelStat) Info() cronv3.TaskInfo {
	return cronv3.TaskInfo{
		Name:      "通道流量记录到数据库",
		Timeout:   10 * time.Second,
		CronSched: cron.Every(20 * time.Second),
	}
}

func (t *tunnelStat) Call(ctx context.Context) error {
	rx, tx := t.mux.Traffic()
	update := bson.M{"$set": bson.M{
		"tunnel_stat.receive_bytes":  rx,
		"tunnel_stat.transmit_bytes": tx,
	}}

	coll := t.db.Broker()
	_, err := coll.UpdateByID(ctx, t.id, update)

	return err
}
