package param

import "github.com/vela-ssoc/vela-common-mb/dynsql"

type MinionBatchRequest struct {
	dynsql.Input
	Cmd     string `json:"cmd"     query:"cmd"     validate:"oneof=resync restart upgrade offline"`
	Keyword string `json:"keyword" query:"keyword"`
}
