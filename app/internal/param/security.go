package param

type SecurityKindRequest struct {
	Kind []string `json:"kind" query:"kind" validate:"lte=20"`
}

type SecurityFileRequest struct {
	Data []string `json:"data" validate:"gte=1,lte=100,dive,hexadecimal"`
}

type SecurityResult struct {
	Count int                 `json:"count"`
	Data  map[string][]string `json:"data"`
}

type SecurityIPRequest struct {
	Data []string `json:"data" validate:"gte=1,lte=100,dive,ip"`
}

type SecurityDNSRequest struct {
	Data []string `json:"data" validate:"gte=1,lte=100,dive,hostname_rfc1123"`
}
