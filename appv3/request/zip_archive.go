package request

type ZipArchiveDownload struct {
	Name    string `json:"name"    query:"name"    validate:"required"`
	Version int64  `json:"version" query:"version"`
}
