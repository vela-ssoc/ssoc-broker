package brokcli

import "time"

type BrokConfig struct {
	DSN         string        `json:"dsn"           validate:"required"`
	MaxOpenConn int           `json:"max_open_conn"`
	MaxIdleConn int           `json:"max_idle_conn"`
	MaxLifeTime time.Duration `json:"max_life_time"`
	MaxIdleTime time.Duration `json:"max_idle_time"`
}

type authRequest struct {
	Secret   string `json:"secret"   validate:"required"` // broker 密钥
	Semver   string `json:"semver"   validate:"required"` // broker 版本号
	Inet     string `json:"inet"     validate:"required"`
	Goos     string `json:"goos"     validate:"required"`
	Goarch   string `json:"goarch"   validate:"required"`
	Hostname string `json:"hostname"`
}

type authResponse struct {
	Code   int         `json:"code"`
	Text   string      `json:"text"`
	Config *BrokConfig `json:"config"`
}

func (r *authResponse) Error() string {
	return r.Text
}

func (r *authResponse) isSucceed() bool {
	return r.Code/100 == 2
}
