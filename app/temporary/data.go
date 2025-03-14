package temporary

import (
	"encoding/json"
	"net"

	"github.com/vela-ssoc/vela-common-mba/ciphertext"
	"github.com/xgfone/ship/v5"
)

type Ident struct {
	Inet    net.IP           `json:"inet"    validate:"required"`
	Inet6   net.IP           `json:"inet6"`
	MAC     net.HardwareAddr `json:"mac"`
	Goos    string           `json:"goos"    validate:"oneof=linux windows darwin"`
	Arch    string           `json:"arch"    validate:"oneof=amd64 386 arm64 arm loong64 riscv64"`
	Edition string           `json:"edition" validate:"semver"`
}

func (i *Ident) decrypt(enc string) error {
	data, err := ciphertext.Decrypt([]byte(enc))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, i)
}

type Claim struct {
	ID    int64  `json:"id,string"` // minion 节点 ID
	Mask  byte   `json:"mask"`      // 协商的掩码
	Token string `json:"token"`     // token
}

func (c Claim) encrypt() (string, error) {
	enc, err := ciphertext.EncryptJSON(c)
	return string(enc), err
}

type StreamIdent struct {
	Mode      Mode            `json:"mode" validate:"required"` // Stream 的连接模式
	Data      json.RawMessage `json:"data"`                     // 内部数据
	validator ship.Validator  // 参数校验器
}

func (i StreamIdent) JSON(v any) error {
	if len(i.Data) != 0 {
		if err := json.Unmarshal(i.Data, v); err != nil {
			return err
		}
	}
	return i.validator.Validate(v)
}

func (i *StreamIdent) decrypt(enc string) error {
	data, err := ciphertext.Decrypt([]byte(enc))
	if err != nil {
		return err
	}
	return json.Unmarshal(data, i)
}
