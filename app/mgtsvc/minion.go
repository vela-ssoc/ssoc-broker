package mgtsvc

import (
	"context"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
)

type MinionService interface {
	LightID(ctx context.Context, id int64) (*param.MinionLight, error)
	Substances(ctx context.Context, light *param.MinionLight) ([]*model.Substance, error)
}

func Minion(qry *query.Query) MinionService {
	return &minionService{qry: qry}
}

type minionService struct{ qry *query.Query }

func (biz *minionService) LightID(ctx context.Context, mid int64) (*param.MinionLight, error) {
	deleted := uint8(model.MSDelete)
	tbl := biz.qry.Minion
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

func (biz *minionService) Substances(ctx context.Context, light *param.MinionLight) ([]*model.Substance, error) {
	if light.Unload {
		return nil, nil
	}

	// 查询该节点的所有标签
	mid, inet := light.ID, light.Inet
	tags := make([]string, 0, 10)
	tagTbl := biz.qry.MinionTag
	_ = tagTbl.WithContext(ctx).
		Distinct(tagTbl.Tag).
		Where(tagTbl.MinionID.Eq(mid)).
		Scan(&tags)

	// 查询节点的配置
	var subIDs []int64
	if len(tags) != 0 {
		effTbl := biz.qry.Effect
		effects, _ := effTbl.WithContext(ctx).
			Where(effTbl.Tag.In(tags...), effTbl.Enable.Is(true)).
			Find()
		subIDs = model.Effects(effects).Exclusion(inet)
	}

	subTbl := biz.qry.Substance
	dao := subTbl.WithContext(ctx).
		Where(subTbl.MinionID.Eq(mid))
	if len(subIDs) != 0 {
		dao.Or(subTbl.ID.In(subIDs...))
	}

	return dao.Find()
}
