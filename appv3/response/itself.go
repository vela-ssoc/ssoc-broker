package response

import (
	"github.com/vela-ssoc/ssoc-broker/config"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
)

type ItselfConfig struct {
	Boot   config.BootConfig  `json:"boot"`
	Config model.BrokerConfig `json:"config"`
}
