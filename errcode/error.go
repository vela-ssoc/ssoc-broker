package errcode

import "github.com/xgfone/ship/v5"

var (
	ErrVictoriaNotConfigured = ship.ErrBadRequest.Newf("没有配置 VictoriaMetrics 地址")
	ErrVictoriaDisabled      = ship.ErrBadRequest.Newf("VictoriaMetrics 未启用")
)

type Errorf interface {
	Fmt(...any) error
}

type formatError string

func (f formatError) Fmt(a ...any) error {
	return ship.ErrBadRequest.Newf(string(f), a...)
}
