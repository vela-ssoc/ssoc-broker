package config

import (
	"context"
	"io"
	"os"

	"github.com/vela-ssoc/ssoc-common-mb/jsonc"
	"github.com/vela-ssoc/vela-common-mba/ciphertext"
)

type HideConfig struct {
	Secret    string   `json:"secret"    validate:"required"`                             // 连接密钥
	Addresses []string `json:"addresses" validate:"required,gte=1,lte=100,dive,required"` // 中心端地址。
}

type Parser interface {
	Parse(ctx context.Context, v any) error
}

type Parsers []Parser

func (ps Parsers) Parse(ctx context.Context, v any) error {
	var err error
	for _, p := range ps {
		if err = p.Parse(ctx, v); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	return err
}

type JSONC string

//goland:noinspection GoUnhandledErrorResult
func (js JSONC) Parse(_ context.Context, v any) error {
	f, err := os.Open(string(js))
	if err != nil {
		return err
	}
	defer f.Close()

	rd := io.LimitReader(f, 1*1024*1024)
	dat, err := io.ReadAll(rd)
	if err != nil {
		return err
	}

	return jsonc.Unmarshal(dat, v)
}

type Binary string

func (bin Binary) Parse(_ context.Context, v any) error {
	return ciphertext.DecryptFile(string(bin), v)
}
