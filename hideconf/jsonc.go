//go:build dev

package hideconf

import (
	"io"
	"os"

	"github.com/vela-ssoc/vela-common-mb/jsonc"
	"github.com/vela-ssoc/vela-common-mb/param/negotiate"
)

const DevMode = true

func Read(file string) (*negotiate.Hide, error) {
	hide := new(negotiate.Hide)
	if file != "" {
		if err := unmarshalJSONC(file, hide); err != nil {
			return nil, err
		}

		return hide, nil
	}

	if err := unmarshalJSONC("broker.jsonc", hide); err == nil {
		return hide, nil
	}

	if err := unmarshalJSONC("broker.json", hide); err == nil {
		return nil, err
	}

	return hide, nil
}

func unmarshalJSONC(name string, v any) error {
	fd, err := os.Open(name)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer fd.Close()

	// 我们使用的 jsonc 不支持流式解析，为了防止恶意大文件造成的 OOM，此处限制
	// 读取文件大小，按照常理和经验，该大小已经足够容纳正常的 jsonc 配置了。
	lr := io.LimitReader(fd, 2<<22) // 8MiB
	data, err := io.ReadAll(lr)
	if err != nil {
		return err
	}

	return jsonc.Unmarshal(data, v)
}
