package subtask

import (
	"context"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
)

type Comparer interface {
	SlowCompare(ctx context.Context, mid int64, inet string, rpt *param.TaskReport) (*param.TaskDiff, []*model.Substance, error)
	FastCompare(ctx context.Context, mid int64, rpt *param.TaskReport, subs []*model.Substance) *param.TaskDiff
}

func Compare() Comparer {
	return &compareService{
		from: "tunnel",
	}
}

type compareService struct {
	from string
}

func (biz *compareService) SlowCompare(ctx context.Context, mid int64, inet string, rpt *param.TaskReport) (*param.TaskDiff, []*model.Substance, error) {
	// 1. 根据 mid 查询所有发布的配置
	effTbl := query.Effect
	subTbl := query.Substance
	tagTbl := query.MinionTag

	tagSQL := tagTbl.WithContext(ctx).Distinct(tagTbl.Tag).Where(tagTbl.MinionID.Eq(mid))
	effs, err := effTbl.WithContext(ctx).Where(effTbl.Enable.Is(true)).
		Where(effTbl.WithContext(ctx).Columns(effTbl.Tag).In(tagSQL)).Find()
	if err != nil {
		return nil, nil, err
	}
	comIDs, subIDs := model.Effects(effs).Exclusion(inet)
	if len(comIDs) != 0 {
		comTbl := query.Compound
		coms, err := comTbl.WithContext(ctx).Where(comTbl.ID.In(comIDs...)).Find()
		if err == nil {
			ids := model.Compounds(coms).SubstanceIDs()
			subIDs = append(subIDs, ids...)
		}
	}

	subDao := subTbl.WithContext(ctx).Where(subTbl.MinionID.Eq(mid))
	if len(subIDs) != 0 {
		subDao.Or(subTbl.ID.In(subIDs...))
	}
	subs, err := subDao.Order(subTbl.ID).Find()
	if err != nil {
		return nil, nil, err
	}

	diff := biz.FastCompare(ctx, mid, rpt, subs)

	return diff, subs, nil
}

func (biz *compareService) FastCompare(_ context.Context, mid int64, rpt *param.TaskReport, subs []*model.Substance) *param.TaskDiff {
	hm := rpt.IDMap()
	removes := make([]int64, 0, 10)
	updates := make([]*param.TaskChunk, 0, 10)
	for _, sub := range subs {
		id := sub.ID
		rp, ok := hm[id]
		delete(hm, id)
		if ok && rp.Hash == sub.Hash {
			continue
		}

		chk := &param.TaskChunk{
			ID:      id,
			Name:    sub.Name,
			Dialect: sub.MinionID == mid,
			Hash:    sub.Hash,
			Chunk:   sub.Chunk,
		}
		updates = append(updates, chk)
	}

	for _, rp := range hm {
		if rp.From == biz.from {
			removes = append(removes, rp.ID)
		}
	}

	diff := &param.TaskDiff{
		Removes: removes,
		Updates: updates,
	}

	return diff
}
