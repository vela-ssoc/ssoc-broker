package service

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/app/subtask"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/integration/alarm"
	"github.com/vela-ssoc/vela-common-mb/integration/cmdb"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
)

type PhaseService interface {
	mlink.NodePhaser
}

func NodeEvent(compare subtask.Comparer, cmdbc cmdb.Client, pool taskpool.Executor, alert alarm.Alerter, slog logback.Logger) PhaseService {
	return &nodeEventService{
		compare: compare,
		pool:    pool,
		cmdbc:   cmdbc,
		slog:    slog,
		alert:   alert,
	}
}

type nodeEventService struct {
	compare subtask.Comparer
	pool    taskpool.Executor
	slog    logback.Logger
	cmdbc   cmdb.Client
	alert   alarm.Alerter
}

func (biz *nodeEventService) Created(id int64, inet string, at time.Time) {
	// 查询状态
	ct := &cmdbTask{
		cli:     biz.cmdbc,
		id:      id,
		inet:    inet,
		slog:    biz.slog,
		timeout: 5 * time.Second,
	}
	biz.pool.Submit(ct)
}

func (biz *nodeEventService) Repeated(id int64, ident gateway.Ident, at time.Time) {
}

func (biz *nodeEventService) Connected(lnk mlink.Linker, ident gateway.Ident, issue gateway.Issue, at time.Time) {
	mid, inet := issue.ID, ident.Inet.String()
	biz.slog.Infof("agent %s(%d) 上线了", inet, mid)

	// 推送 startup
	tsk := subtask.Startup(lnk, mid, biz.slog)
	biz.pool.Submit(tsk)

	task := subtask.SyncTask(lnk, biz.compare, mid, inet, biz.slog)
	biz.pool.Submit(task)

	now := time.Now()
	evt := &model.Event{
		MinionID:  mid,
		Inet:      inet,
		Subject:   "节点下线",
		FromCode:  "minion.offline",
		Msg:       "节点下线",
		Level:     model.ELvlNote,
		SendAlert: false,
		OccurAt:   now,
		CreatedAt: now,
	}
	_ = biz.alert.EventSaveAndAlert(context.Background(), evt)
}

func (biz *nodeEventService) Disconnected(lnk mlink.Linker, ident gateway.Ident, issue gateway.Issue, at time.Time, du time.Duration) {
	mid, inet := issue.ID, ident.Inet.String()
	biz.slog.Warnf("agent %s(%d) 下线了", inet, mid)

	now := time.Now()
	evt := &model.Event{
		MinionID:  mid,
		Inet:      inet,
		Subject:   "节点下线",
		FromCode:  "minion.offline",
		Msg:       "节点下线",
		Level:     model.ELvlNote,
		SendAlert: true,
		OccurAt:   now,
		CreatedAt: now,
	}
	_ = biz.alert.EventSaveAndAlert(context.Background(), evt)
}

type cmdbTask struct {
	cli     cmdb.Client
	id      int64
	inet    string
	slog    logback.Logger
	timeout time.Duration
}

func (ct *cmdbTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), ct.timeout)
	defer cancel()

	if err := ct.cli.FetchAndSave(ctx, ct.id, ct.inet); err != nil {
		ct.slog.Infof("同步 %s 的 cmdb 发生错误：%s", ct.inet, err)
	}
}
