package request

type Sizes struct {
	Size int64 `json:"size" query:"size" validate:"gte=0"`
}

func (s Sizes) Get(defaultVal int64) int64 {
	if s.Size > 0 {
		return s.Size
	}

	return defaultVal
}
