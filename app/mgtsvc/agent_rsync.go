package mgtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
)

func (biz *agentService) RsyncTask(_ context.Context, mids []int64) error {
	for _, mid := range mids {
		task := &rsyncTask{biz: biz, mid: mid}
		biz.pool.Go(task.Run)
	}
	return nil
}

func (biz *agentService) rsyncTask(ctx context.Context, mid int64) error {
	// 查询节点信息
	light, err := biz.mon.LightID(ctx, mid)
	if err != nil {
		return err
	}
	// 2. 同步配置
	return biz.rsync(ctx, light)
}

func (biz *agentService) rsync(ctx context.Context, light *param.MinionLight) error {
	mid, inet := light.ID, light.Inet
	report, err := biz.fetchTaskStatus(ctx, mid) // 拉取最新上报的配置运行状态
	if err != nil {
		return err
	}

	subs, err := biz.mon.Substances(ctx, light) // 查询数据库中关联的配置
	if err != nil {
		return err
	}

	nrp, err := biz.spinRsync(ctx, light, report, subs) // 循环同步

	// 更新上报状态表
	tbl := biz.qry.MinionTask
	_, _ = tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()
	if nrp != nil {
		tasks := nrp.ToModels(mid, inet)
		if len(tasks) != 0 {
			_ = tbl.WithContext(ctx).Create(tasks...)
		}
	}

	return err
}

func (biz *agentService) spinRsync(ctx context.Context, light *param.MinionLight, report *param.TaskReport, subs []*model.Substance) (*param.TaskReport, error) {
	var cycle int
	mid := light.ID
	diff := report.Diff(mid, subs)
	for !diff.NotModified() && cycle < biz.cycle {
		cycle++
		nrp, err := biz.fetchRsync(ctx, mid, diff)
		if err != nil {
			return nil, err
		}
		diff = nrp.Diff(mid, subs)
		report = nrp
	}

	return report, nil
}

func (biz *agentService) fetchRsync(ctx context.Context, mid int64, diff *param.TaskDiff) (*param.TaskReport, error) {
	path := "/api/v1/agent/task/diff"
	ret := new(param.TaskReport)
	if err := biz.lnk.Unicast(ctx, mid, path, diff, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (biz *agentService) fetchTaskStatus(ctx context.Context, mid int64) (*param.TaskReport, error) {
	path := "/api/v1/agent/task/status"
	ret := new(param.TaskReport)
	if err := biz.lnk.Unicast(ctx, mid, path, nil, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

type rsyncTask struct {
	biz *agentService
	mid int64
}

func (rt *rsyncTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_ = rt.biz.rsyncTask(ctx, rt.mid)
}
