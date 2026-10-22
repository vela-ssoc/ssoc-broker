package response

type Records[T any] struct {
	Records []T `json:"records"`
}

func NewRecords[T any](vs []T) Records[T] {
	return Records[T]{Records: vs}
}

type Data[T any] struct {
	Data T `json:"data,omitzero"`
}

func NewData[T any](data T) Data[T] {
	return Data[T]{Data: data}
}
