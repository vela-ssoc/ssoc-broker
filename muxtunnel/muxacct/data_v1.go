package muxacct

import (
	"time"

	"github.com/vela-ssoc/vela-common-mba/ciphertext"
)

// IdentV1 旧版本 agent 节点握手认证时需要携带的信息，
type IdentV1 struct {
	MachineID  string        `json:"machine_id"`                     // 机器 ID
	Inet       string        `json:"inet"       validate:"ip"`       // 内网出口 IP
	GOOS       string        `json:"goos"       validate:"required"` // 操作系统
	Arch       string        `json:"arch"`                           // 操作系统架构
	MAC        string        `json:"mac"`                            // 出口 IP 所在网卡的 MAC 地址
	CPU        int           `json:"cpu"`                            // CPU 核心数
	PID        int           `json:"pid"`                            // 进程 PID
	Workdir    string        `json:"workdir"`                        // 工作目录
	Executable string        `json:"executable"`                     // 执行路径
	Username   string        `json:"username"`                       // 当前操作系统用户名
	Hostname   string        `json:"hostname"`                       // 主机名
	Interval   time.Duration `json:"interval"`                       // 心跳间隔，如果中心端 3 倍心跳仍未收到任何消息，中心端强制断开该连接
	TimeAt     time.Time     `json:"time_at"`                        // agent 当前时间
	Semver     string        `json:"semver"`                         // 节点版本
	Unload     bool          `json:"unload"`                         // 是否开启静默模式，仅在新注册节点时有效
	Unstable   bool          `json:"unstable"`                       // 不稳定版本
	Customized string        `json:"customized"`                     // 定制版本
	Args       []string      `json:"args"`                           // Args 程序运行参数
	Keepalive  bool          `json:"keepalive"`                      // V3 兼容旧版本 Agent 程序（旧版本不会 keepalive）。
}

// Decrypt 认证身份信息解密
func (v *IdentV1) Decrypt(enc []byte) error {
	return ciphertext.DecryptJSON(enc, v)
}

// IssueV1 认证成功后的响应报文信息。
type IssueV1 struct {
	ID     int64  `json:"id"`
	Passwd []byte `json:"passwd"`
}

func (v IssueV1) Encrypt() ([]byte, error) {
	return ciphertext.EncryptJSON(v)
}
