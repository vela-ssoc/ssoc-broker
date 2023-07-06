package tblcmp

import (
	"context"

	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
)

func Find(ctx context.Context, mid int64) (*Record, error) {
	monTbl := query.Minion
	effTbl := query.Effect
	subTbl := query.Substance
	tagTbl := query.MinionTag

	// 1. 查询节点基础信息
	mon, err := monTbl.WithContext(ctx).
		Select(monTbl.ID, monTbl.Inet, monTbl.Status, monTbl.Unload).
		Where(monTbl.ID.Eq(mid)).
		First()
	if err != nil {
		return nil, err
	}

	inet := mon.Inet
	ret := &Record{id: mid, inet: inet, clam: mon.Unload}
	status := mon.Status
	if status == model.MSDelete || status == model.MSInactive || mon.Unload {
		return ret, nil
	}

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
	subs, err := subDao.Order(subTbl.ID).Find()
	if err != nil {
		return nil, err
	}
	ret.subs = subs

	return ret, nil
}
