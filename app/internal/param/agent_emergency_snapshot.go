package param

import (
	"encoding/json"
	"time"
)

type AgentEmergencySnapshotReport struct {
	Type       string          `json:"type"        validate:"required,lte=50"`
	Value      json.RawMessage `json:"value"       validate:"required"`
	ReportedAt time.Time       `json:"reported_at"`
}
