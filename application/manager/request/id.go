package request

type Int64ID struct {
	ID int64 `json:"id" form:"id" query:"id" validate:"required"`
}
