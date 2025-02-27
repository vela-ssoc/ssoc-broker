package mgtsvc

import (
	"context"
	"log/slog"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
)

func (biz *agentService) ReloadTask(_ context.Context, mid, sid int64) error {
	task := &reloadTask{biz: biz, mid: mid, sid: sid}
	biz.pool.Go(task.Run)

	return nil
}

func (biz *agentService) reloadTask(ctx context.Context, mid, sid int64) error {
	light, err := biz.mon.LightID(ctx, mid)
	if err != nil {
		return err
	}
	if light.Unload { // 如果是静默模式就只同步
		biz.log.Warn("节点处于静默模式", slog.Any("minion", light))
		return biz.rsync(ctx, light)
	}

	// 查询要下发的配置
	subTbl := biz.qry.Substance
	sub, err := subTbl.WithContext(ctx).Where(subTbl.ID.Eq(sid)).First()
	if err != nil {
		return biz.rsync(ctx, light)
	}

	// 执行下发配置
	diff := &param.TaskDiff{
		Updates: []*param.TaskChunk{
			{
				ID:      sub.ID,
				Name:    sub.Name,
				Dialect: sub.MinionID == mid,
				Hash:    sub.Hash,
				Chunk:   sub.Chunk,
			},
		},
	}

	_, _ = biz.fetchRsync(ctx, mid, diff)

	// 2. 同步配置
	return biz.rsync(ctx, light)
}

type reloadTask struct {
	biz *agentService
	mid int64
	sid int64
}

func (rt *reloadTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_ = rt.biz.reloadTask(ctx, rt.mid, rt.sid)
}
