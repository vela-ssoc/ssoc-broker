package temporary

import (
	"encoding/binary"
	"encoding/json"

	"github.com/xgfone/ship/v5"
)

type Receive struct {
	minionID  int64          // 来源节点 ID: minion 节点 ID
	opcode    Opcode         // 操作码
	data      []byte         // 数据
	mask      byte           // 加密掩码
	validator ship.Validator // 参数校验器
}

// MinionID 消息的来源 minion ID
func (r Receive) MinionID() int64 {
	return r.minionID
}

// Opcode 获取操作码
func (r Receive) Opcode() Opcode {
	return r.opcode
}

// JSON 将数据绑定
func (r Receive) JSON(v any) error {
	if len(r.data) != 0 {
		if err := json.Unmarshal(r.data, v); err != nil {
			return err
		}
	}
	return r.validator.Validate(v)
}

// unmarshal 处理字节数据
func (r *Receive) unmarshal(raw []byte) error {
	if len(raw) < 2 {
		return &json.SyntaxError{Offset: 2}
	}

	if mask := r.mask; mask != 0 {
		for i := range raw {
			raw[i] ^= mask
		}
	}

	code := binary.BigEndian.Uint16(raw[:2])
	r.opcode = Opcode(code)
	r.data = raw[2:]

	return nil
}
