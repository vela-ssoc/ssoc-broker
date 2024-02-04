package param

import (
	"encoding/json"
	"time"
)

type SharedKey struct {
	Bucket string `json:"bucket" validate:"required,lte=255"`
	Key    string `json:"key"    validate:"lte=255"`
}

type SharedKeyValue struct {
	Bucket   string          `json:"bucket" validate:"required,lte=255"`
	Key      string          `json:"key"    validate:"required,lte=255"`
	Value    json.RawMessage `json:"value"`
	Lifetime time.Duration   `json:"lifetime"`
	Audit    bool            `json:"audit"`
	Reply    bool            `json:"reply"`
}

type SharedKeyIncr struct {
	Bucket string `json:"bucket" validate:"required,lte=255"`
	Key    string `json:"key"    validate:"required,lte=255"`
	N      int64  `json:"n"`
}
