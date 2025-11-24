package clientd

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Config struct {
	Secret    string
	Semver    string
	Addresses []string
}

func (c *Config) preparse() error {
	if c.Secret == "" {
		return errors.New("broker 连接密钥必须填写")
	}
	if c.Semver == "" {
		return errors.New("broker 版本号必须填写")
	}

	uniq := make(map[string]struct{}, 16)
	addrs := make([]string, 0, len(c.Addresses))
	for _, addr := range c.Addresses {
		if _, _, err := net.SplitHostPort(addr); err != nil {
			addr = net.JoinHostPort(addr, "443")
		}
		if _, exists := uniq[addr]; !exists {
			uniq[addr] = struct{}{}
			addrs = append(addrs, addr)
		}
	}
	c.Addresses = addrs
	if len(c.Addresses) == 0 {
		return errors.New("连接地址必须填写")
	}

	return nil
}

type authRequest struct {
	Secret string `json:"secret"`
	Semver string `json:"semver"`
}

type authResponse struct {
	Code     int      `json:"code"`
	Message  string   `json:"message"`
	Database Database `json:"database"`
}

func (ar *authResponse) String() string {
	if err := ar.checkError(); err != nil {
		return err.Error()
	}

	return "agent 认证成功"
}

func (ar *authResponse) checkError() error {
	if ar.Code >= http.StatusOK && ar.Code < http.StatusMultipleChoices {
		return nil
	}

	return fmt.Errorf("broker 认证失败: %s (%d)", ar.Message, ar.Code)
}

func (ar *authResponse) duplicate() bool {
	return ar.Code == http.StatusConflict
}

type Database struct {
	DSN         string        `json:"dsn"           validate:"required"`
	MaxOpenConn int           `json:"max_open_conn"`
	MaxIdleConn int           `json:"max_idle_conn"`
	MaxLifeTime time.Duration `json:"max_life_time"`
	MaxIdleTime time.Duration `json:"max_idle_time"`
}
