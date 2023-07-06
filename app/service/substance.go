package service

import (
	"context"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
)

type SubstanceService interface {
	Compare(ctx context.Context, mid int64, inet string, rt *param.TaskReport) (*param.TaskDiff, error)
}

func Substance() SubstanceService {
	return &substanceService{}
}

type substanceService struct{}

func (biz *substanceService) Compare(ctx context.Context, mid int64, inet string, tr *param.TaskReport) (*param.TaskDiff, error) {
	const tunnelName = "tunnel"
	// 1. 根据 mid 查询所有发布的配置
	effTbl := query.Effect
	subTbl := query.Substance
	tagTbl := query.MinionTag

	tagSQL := tagTbl.WithContext(ctx).Distinct(tagTbl.Tag).Where(tagTbl.MinionID.Eq(mid))
	effs, err := effTbl.WithContext(ctx).Where(effTbl.Enable.Is(true)).
		Where(effTbl.WithContext(ctx).Columns(effTbl.Tag).In(tagSQL)).Find()
	if err != nil {
		return nil, err
	}
	subIDs := model.Effects(effs).Exclusion(inet)

	subDao := subTbl.WithContext(ctx).Where(subTbl.MinionID.Eq(mid))
	if len(subIDs) != 0 {
		subDao.Or(subTbl.ID.In(subIDs...))
	}
	subs, err := subDao.Find()
	if err != nil {
		return nil, err
	}

	// 上报信息转为 map
	rhm := map[int64]*param.TaskStatus{}
	if tr != nil {
		rhm = tr.IDMap()
	}

	removes := make([]int64, 0, 10)
	updates := make([]*param.TaskChunk, 0, 10)
	for _, sub := range subs {
		id := sub.ID
		rpt, ok := rhm[id]
		delete(rhm, id)
		if ok && rpt.Hash == sub.Hash {
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

	for _, rpt := range rhm {
		if rpt.From == tunnelName {
			removes = append(removes, rpt.ID)
		}
	}

	diff := &param.TaskDiff{
		Removes: removes,
		Updates: updates,
	}

	return diff, nil
}
