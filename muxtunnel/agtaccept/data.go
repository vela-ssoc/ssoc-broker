package agtaccept

import (
	"time"

	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/vela-common-mba/ciphertext"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// IdentV1 旧版本 agent 节点握手认证时需要携带的信息，
type IdentV1 struct {
	MachineID  string        `json:"machine_id"`                     // 机器 ID
	Inet       string        `json:"inet"       validate:"ip"`       // 内网出口 IP
	Goos       string        `json:"goos"       validate:"required"` // 操作系统
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
}

// Decrypt 认证身份信息解密
func (v *IdentV1) Decrypt(enc []byte) error {
	return ciphertext.DecryptJSON(enc, v)
}

// IssueV1 认证成功后的响应报文信息。
type IssueV1 struct {
	ID     int64  `json:"id"`
	Passwd []byte `json:"passwd"`
	SID    string `json:"sid"` // 2026 新版新增参数，给 v1 过渡用。
}

func (v IssueV1) Encrypt() ([]byte, error) {
	return ciphertext.EncryptJSON(v)
}

// tunnelSessionDataV1 通道连接后会产生一些数据，这些数据需要在函数之间来回传递。
// 此结构体就是用于存放这些细碎数据的。
type tunnelSessionDataV1 struct {
	ID            bson.ObjectID       `json:"id,omitzero"`
	Peer          muxserver.Peer      `json:"-"`
	Request       *IdentV1            `json:"-"`
	ConnectAt     time.Time           `json:"connect_at,omitzero"`
	DisconnectAt  time.Time           `json:"disconnect_at,omitzero"`
	LocalAddr     string              `json:"local_addr,omitzero"`
	RemoteAddr    string              `json:"remote_addr,omitzero"`
	TunnelLibrary model.TunnelLibrary `json:"tunnel_library,omitzero"`
	ExecuteStat   model.ExecuteStat   `json:"execute_stat,omitzero"`
	TunnelStat    model.TunnelStat    `json:"tunnel_stat,omitzero"`
}

func (d tunnelSessionDataV1) connectedSeconds() uint64 {
	return uint64(d.DisconnectAt.Sub(d.ConnectAt).Seconds())
}
