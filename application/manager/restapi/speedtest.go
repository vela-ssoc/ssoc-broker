package restapi

import (
	"crypto/rand"
	"io"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/vela-ssoc/ssoc-broker/application/manager/request"
	"github.com/xgfone/ship/v5"
)

type Speedtest struct{}

func NewSpeedtest() *Speedtest {
	return &Speedtest{}
}

func (st *Speedtest) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/speedtest").GET(st.download)

	return nil
}

func (st *Speedtest) download(c *ship.Context) error {
	req := new(request.Sizes)
	_ = c.BindQuery(req)
	size := req.Get(100 * 1024 * 1024) // 100M

	nano := time.Now().UnixNano()
	name := strconv.FormatInt(nano, 10)
	dis := mime.FormatMediaType("attachment", map[string]string{"filename": name + ".dat"})
	c.SetRespHeader(ship.HeaderContentDisposition, dis)
	c.SetRespHeader(ship.HeaderContentLength, strconv.FormatInt(size, 10))

	rd := io.LimitReader(rand.Reader, size)

	return c.Stream(http.StatusOK, ship.MIMEOctetStream, rd)
}
