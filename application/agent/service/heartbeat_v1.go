package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type HeartbeatV1 struct {
	db  repository.Database
	log *slog.Logger
}

func NewHeartbeatV1(db repository.Database, log *slog.Logger) *HeartbeatV1 {
	return &HeartbeatV1{
		db:  db,
		log: log,
	}
}

func (h *HeartbeatV1) Ping(ctx context.Context, peer muxserver.Peer) error {
	coll := h.db.Minion()
	id, now := peer.ID(), time.Now()
	update := bson.M{"$set": bson.M{"tunnel_stat.keepalive_at": now}}
	_, err := coll.UpdateByID(ctx, id, update)

	h.log.Debug("节点心跳包", "id", id.Hex(), "info", peer.Info())

	return err
}
