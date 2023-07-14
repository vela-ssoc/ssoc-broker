package mgtsvc

import (
	"context"

	"github.com/vela-ssoc/vela-common-mb/accord"
)

func (biz *agentService) Upgrade(_ context.Context, mids []int64, semver string) error {
	path := "/api/v1/agent/notice/upgrade"
	data := &accord.Upgrade{Semver: semver}

	for _, mid := range mids {
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
