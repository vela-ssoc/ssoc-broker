package param

import "time"

type SharedKey struct {
	Bucket string `json:"bucket" validate:"required,lte=255"`
	Key    string `json:"key"    validate:"required,lte=255"`
}

type SharedKeyValue struct {
	Bucket   string        `json:"bucket" validate:"required,lte=255"`
	Key      string        `json:"key"    validate:"required,lte=255"`
	Value    string        `json:"value"`
	Lifetime time.Duration `json:"lifetime"`
	Audit    bool          `json:"audit"`
}

type SharedKeyIncr struct {
	Bucket   string        `json:"bucket" validate:"required,lte=255"`
	Key      string        `json:"key"    validate:"required,lte=255"`
	N        int64         `json:"n"`
	Lifetime time.Duration `json:"lifetime"`
	Audit    bool          `json:"audit"`
}
