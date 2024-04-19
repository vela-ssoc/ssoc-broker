package mgtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-common-mb-itai/accord"
)

func (biz *agentService) Command(_ context.Context, mids []int64, cmd string) error {
	for _, mid := range mids {
		task := &commandTask{biz: biz, mid: mid, cmd: cmd}
		biz.pool.Submit(task)
	}
	return nil
}

type commandTask struct {
	biz *agentService
	mid int64
	cmd string
}

func (ct *commandTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	dat := &accord.Command{Cmd: ct.cmd}
	path := "/api/v1/agent/notice/command"

	lnk := ct.biz.lnk
	_ = lnk.Oneway(ctx, ct.mid, path, dat)
	if ct.cmd == "offline" || ct.cmd == "restart" {
		lnk.Knockout(ct.mid)
	}
}
