package clientd

import (
	"errors"
	"fmt"
	"net"
	"net/http"
)

type Config struct {
	Secret    string
	Semver    string
	Addresses []string
}

func (c *Config) preparse() error {
	if c.Secret == "" {
		return errors.New("密钥必须填写")
	}
	if c.Semver == "" {
		return errors.New("版本号必须填写")
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
	Code       int        `json:"code"`        // 状态码，2xx 为成功，其他为失败
	Message    string     `json:"message"`     // 错误说明信息
	BootConfig BootConfig `json:"boot_config"` // 认证成功后下发的启动配置
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

	return fmt.Errorf("认证失败: %s (%d)", ar.Message, ar.Code)
}

// isConflict 该节点是否重复上线。
func (ar *authResponse) isConflict() bool {
	return ar.Code == http.StatusConflict
}
