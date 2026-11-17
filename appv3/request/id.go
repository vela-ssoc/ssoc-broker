package request

type StringID struct {
	ID string `json:"id" query:"id" form:"id" param:"id" validate:"required"`
}

type Int64ID struct {
	ID int64 `json:"id" query:"id" form:"id" param:"id" validate:"required"`
}

type Int64IDs struct {
	ID []int64 `json:"id" query:"id" form:"id" param:"id" validate:"gte=1,dive,required"`
}

type Tags struct {
	Tag []string `json:"tag" query:"tag" form:"tag" param:"tag" validate:"gte=1,dive,required"`
}
