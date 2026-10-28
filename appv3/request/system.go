package request

type SystemPID struct {
	PID int32 `json:"pid" query:"pid" form:"pid" param:"pid"`
}
