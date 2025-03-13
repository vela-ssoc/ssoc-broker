package param

import "github.com/vela-ssoc/vela-common-mb/dal/model"

type DeployMinionDownload struct {
	ID         int64        `query:"id"`                            // 客户端安装包 ID
	BrokerID   int64        `query:"broker_id" validate:"required"` // 中心端 ID
	Goos       string       `query:"goos"      validate:"required_without=ID,omitempty,oneof=linux windows darwin"`
	Arch       string       `query:"arch"      validate:"required_without=ID,omitempty,oneof=amd64 386 arm64 arm loong64 riscv64"`
	Version    model.Semver `query:"version"   validate:"omitempty,semver"`
	Unload     bool         `query:"unload"`     // 静默模式
	Unstable   bool         `query:"unstable"`   // 测试版
	Customized string       `query:"customized"` // 定制版标记
	Tags       []string     `query:"tags"      validate:"lte=16,unique,dive,tag"`
}
