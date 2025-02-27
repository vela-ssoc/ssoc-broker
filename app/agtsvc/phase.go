package agtsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vela-ssoc/vela-broker/app/mgtsvc"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/integration/alarm"
	"github.com/vela-ssoc/vela-common-mb/integration/cmdb"
)

type PhaseService interface {
	mlink.NodePhaser
	SetService(svc mgtsvc.AgentService)
}

func Phase(cmdbc cmdb.Client, alert alarm.Alerter, log *slog.Logger) PhaseService {
	return &nodeEventService{
		cmdbc: cmdbc,
		alert: alert,
		pool:  gopool.NewV2(1024),
		log:   log,
	}
}

type nodeEventService struct {
	svc   mgtsvc.AgentService
	cmdbc cmdb.Client
	alert alarm.Alerter
	pool  gopool.Pool
	log   *slog.Logger
}

func (biz *nodeEventService) SetService(svc mgtsvc.AgentService) {
	biz.svc = svc
}

func (biz *nodeEventService) Repeated(id int64, ident gateway.Ident, at time.Time) {
}

func (biz *nodeEventService) Connected(lnk mlink.Linker, ident gateway.Ident, issue gateway.Issue, at time.Time) {
	mid, inet := issue.ID, ident.Inet.String()
	biz.log.Info("Agent 上线", slog.Int64("minion_id", mid), slog.String("inet", inet))

	// 推送 startup 与配置脚本
	ctx := context.Background()
	_ = biz.svc.ReloadStartup(ctx, mid)
	_ = biz.svc.RsyncTask(ctx, []int64{mid})

	msg := fmt.Sprintf("当前 agent 版本：%s", ident.Semver)
	now := time.Now()
	evt := &model.Event{
		MinionID:  mid,
		Inet:      inet,
		Subject:   "节点上线",
		FromCode:  "minion.online",
		Msg:       msg,
		Level:     model.ELvlNote,
		SendAlert: true,
		OccurAt:   now,
		CreatedAt: now,
	}
	_ = biz.alert.EventSaveAndAlert(ctx, evt)
}

func (biz *nodeEventService) Disconnected(lnk mlink.Linker, ident gateway.Ident, issue gateway.Issue, at time.Time, du time.Duration) {
	mid, inet := issue.ID, ident.Inet.String()
	biz.log.Warn("Agent 下线", slog.Int64("minion_id", mid), slog.String("inet", inet))

	msg := fmt.Sprintf("当前 agent 版本：%s", ident.Semver)
	now := time.Now()
	evt := &model.Event{
		MinionID:  mid,
		Inet:      inet,
		Subject:   "节点下线",
		FromCode:  "minion.offline",
		Msg:       msg,
		Level:     model.ELvlMajor,
		SendAlert: true,
		OccurAt:   now,
		CreatedAt: now,
	}

	_ = biz.alert.EventSaveAndAlert(context.Background(), evt)
}
