package subtask

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
)

// func DiffTask(lnk mlink.Linker, compare Comparer, sub *model.Substance, mid int64, inet string, slog logback.Logger) taskpool.Runner {
//	return &diffTask{
//		lnk:     lnk,
//		mid:     mid,
//		inet:    inet,
//		sub:     sub,
//		compare: compare,
//		slog:    slog,
//		timeout: time.Minute,
//		cycle:   5,
//	}
//}

func ThirdDiff(lnk mlink.Linker, diff *param.ThirdDiff) taskpool.Runner {
	return &thirdDiffTask{lnk: lnk, diff: diff}
}

type thirdDiffTask struct {
	lnk  mlink.Linker
	diff *param.ThirdDiff
}

func (td *thirdDiffTask) Run() {
	td.lnk.Broadcast("/api/v1/agent/third/diff", td.diff)
}
