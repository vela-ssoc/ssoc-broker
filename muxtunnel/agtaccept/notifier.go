package agtaccept

import (
	"context"
	"time"

	"github.com/vela-ssoc/ssoc-common/muxserver"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type FailedData struct {
	ID        bson.ObjectID // 数据库 ID
	MachineID string        // 机器码
	Inet      string        // Inet
	Conflict  bool          // 是否重复上线
	Error     error         // 错误信息
}

type ConnectData struct {
	ConnectAt time.Time
}

type DisconnectData struct{}

// Notifier agent 节点连接状态
type Notifier interface {
	// OnFailed 当上线失败时回调。
	OnFailed(ctx context.Context, data *FailedData) error

	OnConnected(ctx context.Context, id bson.ObjectID, info muxserver.PeerInfo) error

	OnDisconnected(ctx context.Context, id bson.ObjectID, info muxserver.PeerInfo, disconnectAt time.Time) error
}

func wrapSafeNotifier(ntf Notifier) *safeNotifier {
	return &safeNotifier{ntf: ntf}
}

type safeNotifier struct {
	ntf Notifier
}

func (s *safeNotifier) OnFailed(ctx context.Context, data *FailedData) error {
	if n := s.ntf; n != nil {
		return n.OnFailed(ctx, data)
	}

	return nil
}

func (s *safeNotifier) OnConnected(ctx context.Context, id bson.ObjectID, info muxserver.PeerInfo) error {
	if n := s.ntf; n != nil {
		return n.OnConnected(ctx, id, info)
	}

	return nil
}

func (s *safeNotifier) OnDisconnected(ctx context.Context, id bson.ObjectID, info muxserver.PeerInfo, disconnectAt time.Time) error {
	if n := s.ntf; n != nil {
		return n.OnDisconnected(ctx, id, info, disconnectAt)
	}

	return nil
}
