package service

import "golang.org/x/time/rate"

type ConnectLimit struct {
	lim *rate.Limiter
}

func NewConnectLimit(ops float64) *ConnectLimit {
	return &ConnectLimit{
		lim: rate.NewLimiter(rate.Limit(ops), int(ops)),
	}
}

func (cl *ConnectLimit) Allowed() bool {
	return cl.lim.Allow()
}
