package service

import (
	"context"
	"log/slog"

	model2 "github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type MinionTagV1 struct {
	pdb polydb.Sourcer
	log *slog.Logger
}

func NewMinionTagV1(pdb polydb.Sourcer, log *slog.Logger) *MinionTagV1 {
	return &MinionTagV1{
		pdb: pdb,
		log: log,
	}
}

func (mt *MinionTagV1) Update(ctx context.Context, node muxproto.Node, adds, dels []string) error {
	rdb := mt.pdb.RDB()
	tbl := rdb.MinionTag
	dao := tbl.WithContext(ctx)

	id := node.Info().IDV1
	tags, err := dao.Where(tbl.MinionID.Eq(id)).Find()
	if err != nil {
		return err
	}
	newTags := model2.MinionTags(tags).Minion(id, adds, dels)

	return rdb.Transaction(func(tx *query.Query) error {
		do := tx.MinionTag.WithContext(ctx)
		if _, err1 := do.Where(tbl.MinionID.Eq(id)).Delete(); err1 != nil {
			return err1
		}
		return do.Create(newTags...)
	})

	// TODO 通知节点重新同步配置。
}
