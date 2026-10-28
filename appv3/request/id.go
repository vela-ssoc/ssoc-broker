package request

type StringID struct {
	ID string `json:"id" query:"id" form:"id" param:"id" validate:"required"`
}
