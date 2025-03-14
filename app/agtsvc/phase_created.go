package agtsvc

import (
	"context"
	"log/slog"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/integration/cmdb"
)

func (biz *nodeEventService) Created(id int64, inet string, at time.Time) {
	// 查询状态
	ct := &cmdbTask{
		cli:  biz.cmdbc,
		id:   id,
		inet: inet,
		log:  biz.log,
	}
	biz.pool.Go(ct.Run)
}

type cmdbTask struct {
	cli  cmdb.Client
	id   int64
	inet string
	log  *slog.Logger
}

func (ct *cmdbTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := ct.cli.FetchAndSave(ctx, ct.id, ct.inet); err != nil {
		ct.log.Info("拉取 cmdb 信息错误", slog.String("inet", ct.inet), slog.Any("error", err))
	}
}
