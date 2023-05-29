package subtask

import (
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
)

func SyncTask(lnk mlink.Linker, compare Comparer, mid int64, inet string, slog logback.Logger) taskpool.Runner {
	return &syncTask{
		lnk:     lnk,
		mid:     mid,
		inet:    inet,
		compare: compare,
		slog:    slog,
		timeout: time.Minute,
		cycle:   5,
	}
}

type syncTask struct {
	lnk     mlink.Linker
	mid     int64
	inet    string
	compare Comparer
	slog    logback.Logger
	timeout time.Duration
	cycle   int
}

func (st *syncTask) Run() {
	ts := &taskSync{
		lnk:     st.lnk,
		mid:     st.mid,
		inet:    st.inet,
		compare: st.compare,
		slog:    st.slog,
		timeout: st.timeout,
		cycle:   st.cycle,
	}
	_ = ts.PullSync()
}

func DiffTask(lnk mlink.Linker, compare Comparer, sub *model.Substance, mid int64, inet string, slog logback.Logger) taskpool.Runner {
	return &diffTask{
		lnk:     lnk,
		mid:     mid,
		inet:    inet,
		sub:     sub,
		compare: compare,
		slog:    slog,
		timeout: time.Minute,
		cycle:   5,
	}
}

type diffTask struct {
	lnk     mlink.Linker
	mid     int64
	inet    string
	sub     *model.Substance
	compare Comparer
	slog    logback.Logger
	timeout time.Duration
	cycle   int
}

func (dt *diffTask) Run() {
	ts := &taskSync{
		lnk:     dt.lnk,
		mid:     dt.mid,
		inet:    dt.inet,
		compare: dt.compare,
		slog:    dt.slog,
		timeout: dt.timeout,
		cycle:   dt.cycle,
	}

	sub := dt.sub
	upd := &param.TaskChunk{
		ID:      sub.ID,
		Name:    sub.Name,
		Dialect: sub.MinionID == dt.mid,
		Hash:    sub.Hash,
		Chunk:   sub.Chunk,
	}
	diff := &param.TaskDiff{Updates: []*param.TaskChunk{upd}}
	_ = ts.DiffSync(diff)
}
