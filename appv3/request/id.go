package request

type StringID struct {
	ID string `json:"id" query:"id" form:"id" param:"id" validate:"required"`
}

type Int64ID struct {
	ID int64 `json:"id,string" query:"id" form:"id" param:"id" validate:"required"`
}
