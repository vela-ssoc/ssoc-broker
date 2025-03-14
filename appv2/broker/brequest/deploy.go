package brequest

import "github.com/vela-ssoc/ssoc-common-mb/dal/model"

type DeployArguments struct {
	ID         int64        `json:"id"         query:"id"`                            // 客户端安装包 ID
	BrokerID   int64        `json:"broker_id"  query:"broker_id" validate:"required"` // 中心端 ID
	Goos       string       `json:"goos"       query:"goos"      validate:"omitempty,oneof=linux windows darwin"`
	Arch       string       `json:"arch"       query:"arch"      validate:"omitempty,oneof=amd64 386 arm64 arm loong64 riscv64"`
	Version    model.Semver `json:"version"    query:"version"   validate:"omitempty,semver"`
	Unload     bool         `json:"unload"     query:"unload"`     // 静默模式
	Unstable   bool         `json:"unstable"   query:"unstable"`   // 测试版
	Customized string       `json:"customized" query:"customized"` // 定制版标记
	Tags       []string     `json:"tags"       query:"tags"      validate:"lte=16,unique,dive,tag"`
}
