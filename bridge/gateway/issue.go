package gateway

import "github.com/vela-ssoc/vela-common-mba/ciphertext"

// Issue 信息
type Issue struct {
	ID     int64  `json:"id"`
	Passwd []byte `json:"passwd"`
}

func (iss Issue) Encrypt() ([]byte, error) {
	return ciphertext.EncryptJSON(iss)
}
