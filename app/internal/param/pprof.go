package param

import "github.com/vela-ssoc/ssoc-common-mb/param/negotiate"

type PprofConfig struct {
	Hide  negotiate.Hide  `json:"hide"`
	Ident negotiate.Ident `json:"ident"`
	Issue negotiate.Issue `json:"issue"`
}
