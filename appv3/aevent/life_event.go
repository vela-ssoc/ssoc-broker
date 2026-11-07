package aevent

import (
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/muxtunnel/apiclient"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/gopool"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type LifeEvent struct {
	pdb  polydb.Sourcer
	cli  apiclient.Client
	pool gopool.Pool
	log  *slog.Logger
}

func NewLifeEvent(pdb polydb.Sourcer, cli apiclient.Client, pool gopool.Pool, log *slog.Logger) *LifeEvent {
	return &LifeEvent{
		pdb:  pdb,
		cli:  cli,
		pool: pool,
		log:  log,
	}
}

func (le *LifeEvent) OnConnected(node muxproto.Node) {
	data := node.Data()
	le.log.Info("节点上线了", "id", data.IDV1)
	push := &onlinePush{life: le, node: node}
	le.pool.Go(push.Run)
}

func (le *LifeEvent) OnDisconnected(node muxproto.Node) {
	data := node.Data()
	le.log.Warn("节点下线了", "id", data.IDV1)
}
