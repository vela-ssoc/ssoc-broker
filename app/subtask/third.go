package subtask

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
)

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
