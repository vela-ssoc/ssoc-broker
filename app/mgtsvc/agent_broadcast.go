package mgtsvc

import (
	"context"
	"time"
)

func (biz *agentService) broadcast(path string, data any) {
	ids := biz.lnk.ConnectIDs()
	for _, mid := range ids {
		task := &messageTask{
			biz:  biz,
			mid:  mid,
			path: path,
			data: data,
		}
		biz.pool.Go(task.Run)
	}
}

type messageTask struct {
	biz  *agentService
	mid  int64
	path string
	data any
}

func (mt *messageTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = mt.biz.lnk.Oneway(ctx, mt.mid, mt.path, mt.data)
}
