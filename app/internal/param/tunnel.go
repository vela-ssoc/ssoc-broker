package param

type TunnelRequest struct {
	Address string `json:"address" query:"address" validate:"required"`
	Skip    bool   `json:"skip"    query:"skip"`
}
