package request

import "golang.org/x/time/rate"

type TunnelLimit struct {
	Limit     float64 `json:"limit"`
	Unlimited bool    `json:"unlimited"`
}

func (l TunnelLimit) Rate() rate.Limit {
	if l.Unlimited {
		return rate.Inf
	}

	return rate.Limit(l.Limit)
}
