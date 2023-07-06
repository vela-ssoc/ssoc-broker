package temporary

import (
	"time"

	"github.com/gorilla/websocket"
)

// Deliver minion 节点消息投递接口约定
type Deliver interface {
	// Unicast 向单个 minion 节点投递消息
	Unicast(int64, Opcode, any) error

	// Multicast 向 1+ 个 minion 节点投递消息
	Multicast([]int64, Opcode, any) error

	// Broadcast 向所有 minion 节点发送消息
	Broadcast(Opcode, any) error
}

type StreamFunc func(*websocket.Conn, error) error

// Handler minion 节点处理器
type Handler interface {
	// Authorize 授权认证处理方法
	Authorize(Ident) (Claim, error)

	// Timeout 消息读取超时时间控制
	Timeout() time.Duration

	// Connect socket 建立连接成功
	Connect(*Conn)

	// Receive 收到 broker 发来的消息
	Receive(*Conn, *Receive)

	// Disconnect broker 断开连接
	Disconnect(*Conn)

	// TokenLookup 通过 token 查询对应的 Conn 连接,
	// 未找到或 token 错误应该返回 nil
	TokenLookup(string) *Conn

	// StreamFunc stream 模式处理器
	StreamFunc(*Conn, StreamIdent) (StreamFunc, error)
}
