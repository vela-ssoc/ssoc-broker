package cronfunc

import (
	"context"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/brokclient"
	"github.com/vela-ssoc/ssoc-common-mb/crontab"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type TunnelStat struct {
	mux brokclient.Muxer
	db  polydb.Sourcer
}

func NewTunnelStat(mux brokclient.Muxer, db polydb.Sourcer) *TunnelStat {
	return &TunnelStat{mux: mux, db: db}
}

func (tnl *TunnelStat) Options() crontab.JobOptions {
	return crontab.JobOptions{
		ID:        "",
		Name:      "统计自身虚拟通道流量数据",
		Sched:     cron.Every(10 * time.Second),
		Timeout:   5 * time.Second,
		Immediate: true,
	}
}

func (tnl *TunnelStat) Execute(ctx context.Context) error {
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
