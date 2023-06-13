package subtask

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
	"gorm.io/gen/field"
)

func Table(lnk mlink.Linker, bid, tid int64, compare Comparer, pool taskpool.Executor, slog logback.Logger) taskpool.Runner {
	return &tableTask{
		tid:     tid,
		bid:     bid,
		size:    200,
		lnk:     lnk,
		compare: compare,
		pool:    pool,
		slog:    slog,
	}
}

type tableTask struct {
	tid     int64
	bid     int64
	size    int // 每批个数
	lnk     mlink.Linker
	compare Comparer
	pool    taskpool.Executor
	slog    logback.Logger
}

func (t *tableTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	tbl := query.SubstanceTask
	dao := tbl.WithContext(ctx).
		Where(tbl.BrokerID.Eq(t.bid), tbl.TaskID.Eq(t.tid), tbl.Executed.Is(false))

	var done bool
	for !done {
		tasks, err := dao.Limit(t.size).Find()
		if err != nil {
			t.slog.Warnf("查询任务错误：%s", err)
			break
		}
		if len(tasks) == 0 {
			done = true
			break
		}

		for _, task := range tasks {
			sync := &taskSync{
				lnk:     t.lnk,
				mid:     task.MinionID,
				inet:    task.Inet,
				compare: t.compare,
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
