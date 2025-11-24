package service

import (
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/channel/agtrpc"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common/gopool"
	"github.com/vela-ssoc/ssoc-common/linkhub"
)

func NewAgentNotifier(qry *query.Query, cli agtrpc.Client, pool gopool.Pool, log *slog.Logger) *AgentNotifier {
	return &AgentNotifier{
		qry:  qry,
		cli:  cli,
		log:  log,
		pool: pool,
	}
}

type AgentNotifier struct {
	qry  *query.Query
	cli  agtrpc.Client
	pool gopool.Pool
	log  *slog.Logger
}

func (an *AgentNotifier) AgentConnected(peer linkhub.Peer) {
	// TODO 每次上线要同步配置脚本
	// 记录上线事件（告警）
}

func (an *AgentNotifier) AgentDisconnected(agentID int64) {
	// 记录下线事件（告警）
}
