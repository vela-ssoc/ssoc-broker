package config

type BootConfig struct {
	Secret    string   `json:"secret"    validate:"required"`                             // 连接密钥
	Addresses []string `json:"addresses" validate:"required,gte=1,lte=100,dive,required"` // 中心端地址。
}
