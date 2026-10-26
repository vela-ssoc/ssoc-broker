package cronfunc

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type TunnelStat struct {
	mux brokclient.Muxer
	db  polydb.Sourcer
}

func NewTunnelStat(mux brokclient.Muxer, db polydb.Sourcer) *TunnelStat {
	return &TunnelStat{mux: mux, db: db}
}

func (tnl *TunnelStat) Info() cronv3.TaskInfo {
	return cronv3.TaskInfo{
		ID:        "afe56e553426c6b3462b67f8e3521001",
		Name:      "统计通道流量数据",
		Immediate: true,
		Timeout:   5 * time.Second,
		CronSched: cron.Every(10 * time.Second),
	}
}

func (tnl *TunnelStat) Call(ctx context.Context) error {
	cfg := tnl.mux.Config()
	rx, tx := tnl.mux.Traffic()
	cumulative, active := tnl.mux.NumStreams()

	filter := bson.M{"slug": cfg.Slug, "status": true}
	update := bson.M{"$set": bson.M{
		"active_streams":             active,
		"tunnel_stat.cumulative":     cumulative,
		"tunnel_stat.receive_bytes":  rx,
		"tunnel_stat.transmit_bytes": tx,
	}}

	coll := tnl.db.MDB().Broker()
	_, err := coll.UpdateOne(ctx, filter, update)

	return err
}
