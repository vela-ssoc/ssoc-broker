package cronjob

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/vela-ssoc/ssoc-common/cronv3"
	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"github.com/vela-ssoc/ssoc-proto/muxproto"
	"github.com/vela-ssoc/ssoc-proto/muxtool"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func NewAgentTunnelStat(db repository.Database, hub muxserver.Huber, basecli muxtool.Client) cronv3.Tasker {
	return &agentTunnelStat{
		db:      db,
		hub:     hub,
		basecli: basecli,
	}
}

type agentTunnelStat struct {
	db      repository.Database
	hub     muxserver.Huber
	basecli muxtool.Client
}

func (t *agentTunnelStat) Info() cronv3.TaskInfo {
	return cronv3.TaskInfo{
		Name:      "agent 通道流量记录到数据库",
		Timeout:   10 * time.Minute,
		CronSched: cron.Every(10 * time.Second),
	}
}

func (t *agentTunnelStat) Call(ctx context.Context) error {
	const batch = 300
	mods := make([]mongo.WriteModel, 0, batch)

	peers := t.hub.Peers()
	for _, p := range peers {
		id := p.ID()
		// t.dial(ctx, id)
		tx, rx := p.MUX().Traffic() // broker 视角统计 agent 流量，互换。
		filter := bson.D{{Key: "_id", Value: id}, {Key: "status", Value: model.MinionStatusOnline}}
		update := bson.M{"$set": bson.M{"tunnel_stat.receive_bytes": rx, "tunnel_stat.transmit_bytes": tx}}
		mod := mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update)
		mods = append(mods, mod)
		if len(mods) >= batch {
			_ = t.bulkWrite(ctx, mods)
			mods = mods[:0]
		}
	}

	return t.bulkWrite(ctx, mods)
}

func (t *agentTunnelStat) bulkWrite(ctx context.Context, mods []mongo.WriteModel) error {
	if len(mods) == 0 {
		return nil
	}

	coll := t.db.Minion()
	opt := options.BulkWrite().SetOrdered(false)
	_, err := coll.BulkWrite(ctx, mods, opt)

	return err
}

func (t *agentTunnelStat) dial(parent context.Context, id bson.ObjectID) error {
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	reqURL := muxproto.ToAgentURL(id.Hex(), "/notfound")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), nil)
	if err != nil {
		return err
	}

	res, err := t.basecli.Do(req)
	if err != nil {
		fmt.Println(">>>", err)
		return err
	}
	defer res.Body.Close()
	fmt.Println(">>>", res.StatusCode)

	return nil
}
