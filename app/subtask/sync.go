package subtask

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/tblcmp"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/logback"
)

const (
	pathTaskDiff   = "/api/v1/agent/task/diff"
	pathTaskStatus = "/api/v1/agent/task/status"
)

type taskSync struct {
	lnk     mlink.Linker
	mid     int64
	slog    logback.Logger
	timeout time.Duration
	cycle   int
}

// DiffSync 推送差异并同步
func (ts *taskSync) DiffSync(diff *param.TaskDiff) error {
	ts.slog.Infof("重启指定配置并同步：%s", diff)
	ctx, cancel := context.WithTimeout(context.Background(), ts.timeout)
	defer cancel()

	rpt, err := ts.postDiff(ctx, diff)
	if err != nil {
		ts.slog.Warnf("推送配置失败，执行中止：%v", err)
		return err
	}
	return ts.spinSync(ctx, rpt)
}

// PullSync 拉取 agent 状态并同步
func (ts *taskSync) PullSync() error {
	ts.slog.Info("拉取节点配置状态并同步")
	ctx, cancel := context.WithTimeout(context.Background(), ts.timeout)
	defer cancel()

	rpt, err := ts.pullStatus(ctx)
	if err != nil {
		ts.slog.Warnf("拉取节点配置状态失败，不再执行同步：%v", err)
		return err
	}
	return ts.spinSync(ctx, rpt)
}

func (ts *taskSync) spinSync(ctx context.Context, rpt *param.TaskReport) error {
	mid := ts.mid
	rec, err := tblcmp.Find(ctx, ts.mid)
	if err != nil {
		ts.slog.Warnf("从数据库比对配置出错：%v", err)
		return err
	}

	var cycle int
	diff := rec.Compare(rpt)
	for cycle < ts.cycle && !diff.NotModified() && err == nil {
		cycle++
		if ret, exx := ts.postDiff(ctx, diff); exx != nil {
			ts.slog.Warnf("第 %d 推送差异出错：%v，差异配置：%s", cycle, err, diff)
		} else {
			rpt = ret
			diff = rec.Compare(ret)
		}
	}

	inet := rec.Inet()
	if err != nil {
		ts.slog.Warnf("向 agent %s(%d)配置同步失败：%s", inet, mid, err)
		return err
	}
	if diff.NotModified() {
		ts.slog.Infof("向 agent %s(%d) 配置同步成功", inet, mid)
	} else {
		ts.slog.Warnf("向 agent %s(%d) 配置同步超过 %d 次数仍未一致", inet, mid, ts.cycle)
	}

	tasks := rpt.ToModels(mid, inet)
	tbl := query.MinionTask
	_, _ = tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()
	if len(tasks) != 0 {
		_ = tbl.WithContext(ctx).Create(tasks...)
	}

	return nil
}

func (ts *taskSync) postDiff(ctx context.Context, diff *param.TaskDiff) (*param.TaskReport, error) {
	ret := new(param.TaskReport)
	if err := ts.lnk.Unicast(ctx, ts.mid, pathTaskDiff, diff, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

func (ts *taskSync) pullStatus(ctx context.Context) (*param.TaskReport, error) {
	ret := new(param.TaskReport)
	if err := ts.lnk.Unicast(ctx, ts.mid, pathTaskStatus, nil, ret); err != nil {
		return nil, err
	}
	return ret, nil
}
