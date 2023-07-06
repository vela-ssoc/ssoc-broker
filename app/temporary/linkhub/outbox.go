package linkhub

import "github.com/vela-ssoc/vela-broker/app/temporary"

// outboxMsg 发件箱消息
type outboxMsg struct {
	MinionID  int64            // 接收节点 ID
	Broadcast bool             // 是否是广播
	Opcode    temporary.Opcode // 操作码
	Data      any              // 数据
}

// IsBroadcast 是否是广播消息
func (m outboxMsg) IsBroadcast() bool {
	return m.Broadcast && m.MinionID <= 0
}
