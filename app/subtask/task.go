package subtask

import (
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/logback"
)

func SyncTask(lnk mlink.Linker, mid int64, slog logback.Logger) gopool.Runner {
	return &syncTask{
		lnk:     lnk,
		mid:     mid,
		slog:    slog,
		timeout: time.Minute,
		cycle:   5,
	}
}

type syncTask struct {
	lnk     mlink.Linker
	mid     int64
	slog    logback.Logger
	timeout time.Duration
	cycle   int
}

func (st *syncTask) Run() {
	ts := &taskSync{
		lnk:     st.lnk,
		mid:     st.mid,
		slog:    st.slog,
		timeout: st.timeout,
		cycle:   st.cycle,
	}
	_ = ts.PullSync()
}

func DiffTask(lnk mlink.Linker, sub *model.Substance, mid int64, slog logback.Logger) gopool.Runner {
	return &diffTask{
		lnk:     lnk,
		mid:     mid,
		sub:     sub,
		slog:    slog,
		timeout: time.Minute,
		cycle:   5,
	}
}

type diffTask struct {
	lnk     mlink.Linker
	mid     int64
	sub     *model.Substance
	slog    logback.Logger
	timeout time.Duration
	cycle   int
}

func (dt *diffTask) Run() {
	ts := &taskSync{
		lnk:     dt.lnk,
		mid:     dt.mid,
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
