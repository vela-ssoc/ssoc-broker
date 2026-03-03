package service

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type SysInfoV1 struct {
	db  repository.Database
	log *slog.Logger
}

func NewSysInfoV1(db repository.Database, log *slog.Logger) *SysInfoV1 {
	return &SysInfoV1{
		db:  db,
		log: log,
	}
}

// Report 上报系统信息。
func (si *SysInfoV1) Report(ctx context.Context, info *model.MinionSysInfo, peer muxserver.Peer) error {
	id := peer.ID()

	coll := si.db.Minion()
	update := bson.M{"$set": bson.M{"sys_info": info}}
	_, err := coll.UpdateByID(ctx, id, update)

	return err
}
