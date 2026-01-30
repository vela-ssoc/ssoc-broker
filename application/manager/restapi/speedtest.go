package restapi

import (
	"crypto/rand"
	"net/http"

	"github.com/xgfone/ship/v5"
)

type Speedtest struct{}

func NewSpeedtest() *Speedtest {
	return &Speedtest{}
}

func (st *Speedtest) download(c *ship.Context) error {
	return c.Stream(http.StatusOK, ship.MIMEOctetStream, rand.Reader)
}
