package param

type TagRequest struct {
	Add []string `json:"add" validate:"lte=50,unique,dive,tag"`
	Del []string `json:"del" validate:"lte=50,unique,dive,tag"`
}

func (r TagRequest) Empty() bool {
	return len(r.Del) == 0 && len(r.Add) == 0
}
