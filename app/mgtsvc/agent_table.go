package mgtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-common-mb-itai/dal/query"
	"gorm.io/gen/field"
)

func (biz *agentService) TableTask(_ context.Context, tid int64) error {
	bid := biz.lnk.Link().Ident().ID
	go biz.scanTableTask(bid, tid)
	return nil
}

func (biz *agentService) scanTableTask(bid, tid int64) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	tbl := query.SubstanceTask
	dao := tbl.WithContext(ctx).
		Distinct(tbl.MinionID).
		Where(tbl.TaskID.Eq(tid), tbl.BrokerID.Eq(bid), tbl.Executed.Is(false)).
		Order(tbl.MinionID).
		Limit(200)

	var lastID int64
	for {
		var mids []int64
		_ = dao.Where(tbl.MinionID.Gt(lastID)).Scan(&mids)
		size := len(mids)
		if size == 0 {
			break
		}
		lastID = mids[size-1]

		for _, mid := range mids {
			task := &tableTask{
				biz: biz,
				bid: bid,
				tid: tid,
				mid: mid,
			}
			biz.pool.Submit(task)
		}
	}
}

type tableTask struct {
	biz *agentService
	bid int64
	tid int64
	mid int64
}

func (st *tableTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := st.biz.rsyncTask(ctx, st.mid)

	if ctx.Err() != nil {
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
	}

	tbl := query.SubstanceTask
	assigns := []field.AssignExpr{
		tbl.Executed.Value(true),
	}
	if err != nil {
		assigns = append(assigns, tbl.Failed.Value(true))
		assigns = append(assigns, tbl.Reason.Value(err.Error()))
	}

	_, _ = tbl.WithContext(ctx).
		Where(
			tbl.BrokerID.Eq(st.bid),
			tbl.TaskID.Eq(st.tid),
			tbl.MinionID.Eq(st.mid),
			tbl.Executed.Is(false),
		).UpdateSimple(assigns...)
}
