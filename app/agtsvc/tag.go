package agtsvc

import (
	"context"
	"errors"

	"github.com/vela-ssoc/ssoc-broker/app/mgtsvc"
	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"gorm.io/gorm/clause"
)

type TagService interface {
	Update(ctx context.Context, mid int64, creates, deletes []string) error
}

func Tag(qry *query.Query, svc mgtsvc.AgentService) TagService {
	return &tagService{qry: qry, svc: svc}
}

type tagService struct {
	qry *query.Query
	svc mgtsvc.AgentService
}

func (biz *tagService) Update(ctx context.Context, mid int64, creates, deletes []string) error {
	monTbl := biz.qry.Minion
	mon, err := monTbl.WithContext(ctx).
		Select(monTbl.Status, monTbl.BrokerID, monTbl.Inet).
		Where(monTbl.ID.Eq(mid)).
		First()
	if err != nil {
		return err
	}
	if mon.Status == model.MSDelete {
		return errors.New("节点已删除")
	}

	tbl := biz.qry.MinionTag
	// 查询现有的 tags
	olds, err := tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Find()
	if err != nil {
		return err
	}
	news := model.MinionTags(olds).Minion(mid, deletes, creates)
	err = biz.qry.Transaction(func(tx *query.Query) error {
		table := tx.WithContext(ctx).MinionTag
		if _, exx := table.Where(tbl.MinionID.Eq(mid)).
			Delete(); exx != nil || len(news) == 0 {
			return exx
		}
		return table.Clauses(clause.OnConflict{DoNothing: true}).
			CreateInBatches(news, 100)
	})
	if err != nil {
		return err
	}

	_ = biz.svc.RsyncTask(ctx, []int64{mid})

	return nil
}
