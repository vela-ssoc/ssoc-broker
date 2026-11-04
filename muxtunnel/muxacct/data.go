package muxacct

import (
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
)

type sessionDataV1 struct {
	Ident          IdentV1
	Minion         *model.Minion
	ConnectedAt    time.Time
	DisconnectedAt time.Time
}
