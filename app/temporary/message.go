package temporary

import (
	"encoding/binary"
	"encoding/json"
)

type Message struct {
	Opcode Opcode // 操作码
	Data   any    // 原始数据
	mask   byte   // 加密掩码
}

// marshal 消息序列化为 []byte
func (m Message) marshal() ([]byte, error) {
	var ret []byte
	if m.Data == nil {
		ret = make([]byte, 2)
		binary.BigEndian.PutUint16(ret, uint16(m.Opcode))
	} else {
		data, err := json.Marshal(m.Data)
		if err != nil {
			return nil, err
		}
		ret = make([]byte, 2+len(data))
		binary.BigEndian.PutUint16(ret, uint16(m.Opcode))
		copy(ret[2:], data)
	}

	if mask := m.mask; mask != 0 {
		for i := range ret {
			ret[i] ^= mask
		}
	}

	return ret, nil
}
