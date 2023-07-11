package gateway

import (
	"net"
	"time"

	"github.com/vela-ssoc/vela-common-mba/ciphertext"
)

// Ident minion 节点握手认证时需要携带的信息
type Ident struct {
	Semver     string        `json:"semver"`     // 节点版本
	Inet       net.IP        `json:"inet"`       // 内网出口 IP
	MAC        string        `json:"mac"`        // 出口 IP 所在网卡的 MAC 地址
	Goos       string        `json:"goos"`       // 操作系统 runtime.GOOS
	Arch       string        `json:"arch"`       // 操作系统架构 runtime.GOARCH
	CPU        int           `json:"cpu"`        // CPU 核心数
	PID        int           `json:"pid"`        // 进程 PID
	Workdir    string        `json:"workdir"`    // 工作目录
	Executable string        `json:"executable"` // 执行路径
	Username   string        `json:"username"`   // 当前操作系统用户名
	Hostname   string        `json:"hostname"`   // 主机名
	Interval   time.Duration `json:"interval"`   // 心跳间隔
	Unload     bool          `json:"unload"`     // 是否开启静默模式，仅对新注册的节点有效
	TimeAt     time.Time     `json:"time_at"`    // 时间

	// Encrypt 是否加密传输。
	// Deprecated: 后续将删除该字段，默认所有数据加密传输。
	Encrypt bool `json:"encrypt"` // 是否支持加密
}

// Decrypt 认证身份信息解密
func (ide *Ident) Decrypt(enc []byte) error {
	return ciphertext.DecryptJSON(enc, ide)
}
