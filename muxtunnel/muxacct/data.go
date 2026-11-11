package muxacct

import (
	"time"

	model2 "github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
)

type sessionDataV1 struct {
	ConnectedAt        time.Time
	BrokerConfigParams model.BrokerConfigParams
	Ident              IdentV1
	Minion             *model2.Minion
	DisconnectedAt     time.Time
}
