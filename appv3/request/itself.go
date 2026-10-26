package request

import "time"

type ItselfExit struct {
	Seconds int `json:"seconds" query:"seconds" validate:"gte=0"`
}

func (ie ItselfExit) Duration() time.Duration {
	return time.Duration(ie.Seconds) * time.Second
}
