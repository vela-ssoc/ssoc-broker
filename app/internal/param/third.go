package param

type ThirdDownload struct {
	Name string `json:"name" query:"name" validate:"required,lte=255"`
	Hash string `json:"hash" query:"hash"`
}

type ThirdDiff struct {
	Name  string `json:"name"`
	Event string `json:"event"`
}
