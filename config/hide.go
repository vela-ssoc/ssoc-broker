package config

type Hide struct {
	Secret    string   `json:"secret"    validate:"required,lte=1000"`           // 连接认证密钥
	Protocol  string   `json:"protocol"  validate:"omitempty,oneof=smux yamux"`  // 通信协议
	Addresses []string `json:"addresses" validate:"gte=1,lte=100,dive,required"` // manager 地址
}
