package subtask

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"gorm.io/gen/field"
)

func Table(lnk mlink.Linker, bid, tid int64, pool gopool.Executor, slog logback.Logger) gopool.Runner {
	return &tableTask{
		tid:  tid,
		bid:  bid,
		size: 200,
		lnk:  lnk,
		pool: pool,
		slog: slog,
	}
}

type tableTask struct {
	tid  int64
	bid  int64
	size int // 每批个数
	lnk  mlink.Linker
	pool gopool.Executor
	slog logback.Logger
}

func (t *tableTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	tbl := query.SubstanceTask
	dao := tbl.WithContext(ctx).
		Where(tbl.BrokerID.Eq(t.bid), tbl.TaskID.Eq(t.tid), tbl.Executed.Is(false)).
		Order(tbl.MinionID)

	var offset int
	var done bool
	for !done {
		tasks, err := dao.Offset(offset).Limit(t.size).Find()
		if err != nil {
			t.slog.Warnf("查询任务错误：%s", err)
			break
		}
		length := len(tasks)
		if done = length == 0; done {
			break
		}
		offset += length

		for _, task := range tasks {
			sync := &taskSync{
				lnk:     t.lnk,
				mid:     task.MinionID,
				slog:    t.slog,
				timeout: time.Minute,
				cycle:   3,
			}
			tst := &tableSubTask{
				sync: sync,
				tid:  t.tid,
				bid:  t.bid,
				mid:  task.MinionID,
			}
			t.pool.Submit(tst)
		}
	}
}

type tableSubTask struct {
	tid  int64
	bid  int64
	mid  int64
	sync *taskSync
}

func (t *tableSubTask) Run() {
	err := t.sync.PullSync()
	tbl := query.SubstanceTask
	assigns := []field.AssignExpr{tbl.Executed.Value(true)}
	if err != nil {
		assigns = append(assigns, tbl.Failed.Value(true))
		assigns = append(assigns, tbl.Reason.Value(err.Error()))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = tbl.WithContext(ctx).
		Where(tbl.MinionID.Eq(t.mid), tbl.BrokerID.Eq(t.bid), tbl.TaskID.Eq(t.tid), tbl.Executed.Is(false)).
		UpdateColumnSimple(assigns...)
}
