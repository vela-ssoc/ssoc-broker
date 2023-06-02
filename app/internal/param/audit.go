package param

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/vela-ssoc/vela-common-mb/dal/model"
)

type AuditRiskRequest struct {
	// Class 风险类型
	// ["暴力破解", "病毒事件", "弱口令", "数据爬虫", "蜜罐应用", "web 攻击", "监控事件", "登录事件"]
	Class      string          `json:"class"       validate:"required"`
	Inet       string          `json:"inet"        validate:"required"` // 节点 IPv4
	Level      model.RiskLevel `json:"level"       validate:"required"` // 风险级别
	Payload    string          `json:"payload"`                         // 攻击载荷
	Subject    string          `json:"subject"     validate:"required"` // 风险事件主题
	LocalIP    string          `json:"local_ip"`                        // 本地 IP
	LocalPort  int             `json:"local_port"`                      // 本地端口
	RemoteIP   string          `json:"remote_ip"`                       // 远程 IP
	RemotePort int             `json:"remote_port"`                     // 远程端口
	FromCode   string          `json:"from_code"`                       // 来源模块
	Region     string          `json:"region"`                          // IP 归属地
	Reference  string          `json:"reference"`                       // 参考引用
	Alert      bool            `json:"alert"`                           // 是否需要发送告警
	Time       time.Time       `json:"time"`                            // 风险产生的时间
}

func (r AuditRiskRequest) Model(minionID int64) *model.Risk {
	var secret string
	if r.Alert {
		buf := make([]byte, 10)
		_, _ = rand.Read(buf)
		secret = hex.EncodeToString(buf)
	}
	if r.Time.IsZero() {
		r.Time = time.Now()
	}

	return &model.Risk{
		MinionID:   minionID,
		Inet:       r.Inet,
		RiskType:   r.Class,
		Level:      r.Level,
		Payload:    r.Payload,
		Subject:    r.Subject,
		LocalIP:    r.LocalIP,
		LocalPort:  r.LocalPort,
		RemoteIP:   r.RemoteIP,
		RemotePort: r.RemotePort,
		FromCode:   r.FromCode,
		Region:     r.Region,
		Reference:  r.Reference,
		SendAlert:  r.Alert,
		Secret:     secret,
		OccurAt:    r.Time,
		Status:     model.RSUnprocessed,
	}
}
