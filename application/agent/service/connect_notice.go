package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/vela-ssoc/ssoc-broker/muxtunnel/agtaccept"
	"github.com/vela-ssoc/ssoc-common/muxserver"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type ConnectNotice struct {
	log           *slog.Logger
	failedCnt     *metrics.Counter // 上线失败，含重复上线
	conflictCnt   *metrics.Counter // 重复上线
	succeedCnt    *metrics.Counter // 上线成功
	disconnectCnt *metrics.Counter // 下线
}

func NewConnectNotice(log *slog.Logger) *ConnectNotice {
	return &ConnectNotice{
		failedCnt:     metrics.NewCounter("agent_tunnel_connect_fails"),
		conflictCnt:   metrics.NewCounter("agent_tunnel_connect_conflicts"),
		succeedCnt:    metrics.NewCounter("agent_tunnel_connect_succeeds"),
		disconnectCnt: metrics.NewCounter("agent_tunnel_disconnects"),
		log:           log,
	}
}

func (cn *ConnectNotice) OnFailed(ctx context.Context, data *agtaccept.FailedData) error {
	cn.failedCnt.Inc()
	if data.Conflict {
		cn.conflictCnt.Inc()
	}

	return nil
}

func (cn *ConnectNotice) OnConnected(ctx context.Context, id bson.ObjectID, info muxserver.PeerInfo) error {
	cn.succeedCnt.Inc()
	// TODO 异步通知节点同步配置
	return nil
}

func (cn *ConnectNotice) OnDisconnected(ctx context.Context, id bson.ObjectID, info muxserver.PeerInfo, disconnectAt time.Time) error {
	cn.disconnectCnt.Inc()
	return nil
}
