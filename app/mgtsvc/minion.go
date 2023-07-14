package mgtsvc

import (
	"context"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
)

type MinionService interface {
	LightID(ctx context.Context, id int64) (*param.MinionLight, error)
	Substances(ctx context.Context, mid int64, inet string) ([]*model.Substance, error)
}

func Minion() MinionService {
	return &minionService{}
}

type minionService struct{}

func (biz *minionService) LightID(ctx context.Context, mid int64) (*param.MinionLight, error) {
	deleted := uint8(model.MSDelete)
	tbl := query.Minion
	mon, err := tbl.WithContext(ctx).
		Select(tbl.ID, tbl.BrokerID, tbl.Unload, tbl.Inet).
		Where(tbl.ID.Eq(mid), tbl.Status.Neq(deleted)).
		First()
	if err != nil {
		return nil, err
	}
	ret := &param.MinionLight{
		ID:     mid,
		Inet:   mon.Inet,
		Unload: mon.Unload,
	}
	return ret, nil
}

func (biz *minionService) Substances(ctx context.Context, mid int64, inet string) ([]*model.Substance, error) {
	// 查询该节点的所有标签
	tags := make([]string, 0, 10)
	tagTbl := query.MinionTag
	_ = tagTbl.WithContext(ctx).
		Distinct(tagTbl.Tag).
		Where(tagTbl.MinionID.Eq(mid)).
		Scan(&tags)

	// 查询节点的配置
	var subIDs []int64
	if len(tags) != 0 {
		effTbl := query.Effect
		effects, _ := effTbl.WithContext(ctx).
			Where(effTbl.Tag.In(tags...)).
			Find()
		subIDs = model.Effects(effects).Exclusion(inet)
	}

	subTbl := query.Substance
	dao := subTbl.WithContext(ctx).
		Where(subTbl.MinionID.Eq(mid))
	if len(subIDs) != 0 {
		dao.Or(subTbl.ID.In(subIDs...))
	}

	return dao.Find()
}
