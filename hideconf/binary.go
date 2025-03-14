//go:build !dev

package hideconf

import (
	"os"

	"github.com/vela-ssoc/ssoc-common-mb/param/negotiate"
	"github.com/vela-ssoc/vela-common-mba/ciphertext"
)

const DevMode = false

func Read(file string) (*negotiate.Hide, error) {
	if file == "" {
		file = os.Args[0]
	}

	hide := new(negotiate.Hide)
	if err := ciphertext.DecryptFile(file, hide); err != nil {
		return nil, err
	}

	return hide, nil
}
