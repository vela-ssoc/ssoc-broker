package config

import (
	"fmt"
	"log/slog"
)

type Config struct {
	Secret    secret   `json:"secret"    yaml:"secret"    validate:"required"`                           // broker 密钥
	Semver    string   `json:"semver"    yaml:"semver"    validate:"required"`                           // 版本号，例如：1.2.3-beta
	Addresses []string `json:"addresses" yaml:"addresses" validate:"gte=1,lte=100,unique,dive,required"` // manager 地址
}

type secret string

func (s secret) LogValue() slog.Value {
	size := len(s)
	if size <= 16 {
		return slog.StringValue("******")
	}
	str := s[:4] + "******" + s[size-4:]

	return slog.StringValue(string(str))
}

func (c Config) String() string {
	return fmt.Sprintf("版本号：%s 地址：%s 密钥：%s", c.Semver, c.Addresses, c.Secret.LogValue())
}
