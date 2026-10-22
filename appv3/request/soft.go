package request

type SoftExit struct {
	Seconds int `json:"seconds" query:"seconds" validate:"gte=1"`
}
