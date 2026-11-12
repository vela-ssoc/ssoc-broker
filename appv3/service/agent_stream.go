package service

import (
	"context"
	"log/slog"

	model2 "github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	"github.com/vela-ssoc/vela-tunnel/muxconn"
)

type AgentStream struct {
	pdb polydb.Sourcer
	hub muxproto.Huber
	log *slog.Logger
}

func NewAgentStream(pdb polydb.Sourcer, hub muxproto.Huber, log *slog.Logger) *AgentStream {
	return &AgentStream{pdb: pdb, hub: hub, log: log}
}

func (bs *AgentStream) Substreams(id int64) ([]muxconn.Streamer, bool) {
	host := muxproto.Int64ToHost(id)
	node := bs.hub.Get(host)
	if node == nil {
		return nil, false
	}

	mux := node.MUX()
	streams := mux.Streams()

	return streams, true
}

func (bs *AgentStream) Close(ctx context.Context, id int64) error {
	host := muxproto.Int64ToHost(id)
	if node := bs.hub.Get(host); node != nil {
		mux := node.MUX()
		_ = mux.Close()
		return nil
	}

	// 连接池中不存在，检查是否数据库状态是否在线，如果在线说明异常退出，
	// 中心端没有来得及标记为下线，状态数据不一致要进行纠错。
	qry := bs.pdb.RDB()
	tbl := qry.Minion
	dao := tbl.WithContext(ctx)

	mon, err := dao.Where(tbl.ID.Eq(id)).First()
	if err != nil {
		return err
	} else if mon.Status != model2.MSOnline {
		return nil
	}

	offline := uint8(model2.MSOffline)
	_, err = dao.Where(tbl.ID.Eq(id)).
		UpdateColumnSimple(tbl.Status.Value(offline))

	return err
}
