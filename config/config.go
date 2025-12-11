package config

type Config struct {
	Secret    string   `json:"secret"    yaml:"secret"    validate:"required"`                    // broker 密钥
	Semver    string   `json:"semver"    yaml:"semver"    validate:"required"`                    // 版本号，例如：1.2.3-beta
	Addresses []string `json:"addresses" yaml:"addresses" validate:"gte=1,lte=100,dive,required"` // manager 地址
}
