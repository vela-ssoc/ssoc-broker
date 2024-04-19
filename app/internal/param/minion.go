package param

import (
	"fmt"

	"github.com/vela-ssoc/vela-common-mb-itai/dynsql"
)

type MinionBatchRequest struct {
	dynsql.Input
	Cmd     string `json:"cmd"     query:"cmd"     validate:"oneof=resync restart upgrade offline"`
	Keyword string `json:"keyword" query:"keyword"`
}

type MinionLight struct {
	ID     int64
	Inet   string
	Unload bool
}

func (m MinionLight) String() string {
	return fmt.Sprintf("%s(%d)", m.Inet, m.ID)
}
