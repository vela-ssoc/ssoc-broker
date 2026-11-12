package request

type AgentSubstreamKill struct {
	Int64ID
	StreamID uint64 `json:"stream_id,string" query:"stream_id"`
}
