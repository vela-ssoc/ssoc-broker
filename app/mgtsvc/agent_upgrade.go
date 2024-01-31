package mgtsvc

import (
	"context"

	"github.com/vela-ssoc/vela-common-mb/accord"
)

func (biz *agentService) Upgrade(_ context.Context, req *accord.Upgrade) error {
	path := "/api/v1/agent/notice/upgrade"
	data := &accord.Upgrade{Semver: req.Semver, Customized: req.Customized}

	for _, mid := range req.ID {
		task := &messageTask{
			biz:  biz,
			mid:  mid,
			path: path,
			data: data,
		}
		biz.pool.Submit(task)
	}

	return nil
}
