package param

import "github.com/vela-ssoc/vela-broker/bridge/telecom"

type PprofConfig struct {
	Hide  telecom.Hide  `json:"hide"`
	Ident telecom.Ident `json:"ident"`
	Issue telecom.Issue `json:"issue"`
}
