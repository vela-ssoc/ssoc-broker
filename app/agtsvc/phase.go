package agtsvc

import (
	"context"
	"time"

	"github.com/vela-ssoc/vela-broker/app/mgtsvc"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/integration/alarm"
	"github.com/vela-ssoc/vela-common-mb/integration/cmdb"
	"github.com/vela-ssoc/vela-common-mb/logback"
)

type PhaseService interface {
	mlink.NodePhaser
	SetService(svc mgtsvc.AgentService)
}

func Phase(cmdbc cmdb.Client, alert alarm.Alerter, slog logback.Logger) PhaseService {
	return &nodeEventService{
		cmdbc: cmdbc,
		alert: alert,
		slog:  slog,
		pool:  gopool.New(64, 64, time.Minute),
	}
}

type nodeEventService struct {
	svc   mgtsvc.AgentService
	cmdbc cmdb.Client
	alert alarm.Alerter
	slog  logback.Logger
	pool  gopool.Executor
}

func (biz *nodeEventService) SetService(svc mgtsvc.AgentService) {
	biz.svc = svc
}

func (biz *nodeEventService) Repeated(id int64, ident gateway.Ident, at time.Time) {
}

func (biz *nodeEventService) Connected(lnk mlink.Linker, ident gateway.Ident, issue gateway.Issue, at time.Time) {
	mid, inet := issue.ID, ident.Inet.String()
	biz.slog.Infof("agent %s(%d) 上线了", inet, mid)

	// 推送 startup 与配置脚本
	ctx := context.Background()
	_ = biz.svc.ReloadStartup(ctx, mid)
	_ = biz.svc.RsyncTask(ctx, []int64{mid})

	now := time.Now()
	evt := &model.Event{
		MinionID:  mid,
		Inet:      inet,
		Subject:   "节点上线",
		FromCode:  "minion.online",
		Msg:       "节点上线",
		Level:     model.ELvlNote,
		SendAlert: true,
		OccurAt:   now,
		CreatedAt: now,
	}
	task := &eventTask{
		alert: biz.alert,
		evt:   evt,
	}
	biz.pool.Submit(task)
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
		Level:     model.ELvlMajor,
		SendAlert: true,
		OccurAt:   now,
		CreatedAt: now,
	}

	task := &eventTask{
		alert: biz.alert,
		evt:   evt,
	}
	biz.pool.Submit(task)
}

type eventTask struct {
	alert alarm.Alerter
	evt   *model.Event
}

func (et *eventTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	_ = et.alert.EventSaveAndAlert(ctx, et.evt)
}
