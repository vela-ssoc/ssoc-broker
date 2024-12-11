package agtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-common-mb/integration/cmdb"
	"github.com/vela-ssoc/vela-common-mb/logback"
)

func (biz *nodeEventService) Created(id int64, inet string, at time.Time) {
	// 查询状态
	ct := &cmdbTask{
		cli:  biz.cmdbc,
		id:   id,
		inet: inet,
		slog: biz.slog,
	}
	biz.pool.Go(ct.Run)
}

type cmdbTask struct {
	cli  cmdb.Client
	id   int64
	inet string
	slog logback.Logger
}

func (ct *cmdbTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := ct.cli.FetchAndSave(ctx, ct.id, ct.inet); err != nil {
		ct.slog.Infof("同步 %s 的 cmdb 发生错误：%s", ct.inet, err)
	}
}
