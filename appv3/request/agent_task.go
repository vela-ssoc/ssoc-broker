package request

type AgentTaskReload struct {
	ID          int64 `json:"id"`
	SubstanceID int64 `json:"substance_id"`
}
