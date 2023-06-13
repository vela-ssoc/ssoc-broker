package param

type SecurityKindRequest struct {
	Kind []string `json:"kind" query:"kind" validate:"lte=20"`
}

type SecurityFileRequest struct {
	Data []string `json:"data" validate:"gte=1,lte=100,dive,hexadecimal"`
}

type SecurityFileResult struct {
	Count int                 `json:"count"`
	Data  map[string][]string `json:"data"`
}
