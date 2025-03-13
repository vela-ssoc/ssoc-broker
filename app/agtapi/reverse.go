package agtapi

import (
	"io/fs"
	"mime"
	"net/http"
	"strconv"

	"github.com/vela-ssoc/ssoc-broker/app/internal/param"
	"github.com/vela-ssoc/ssoc-broker/app/route"
	"github.com/xgfone/ship/v5"
)

func Reverse(elkeidFS fs.FS) route.Router {
	return &reverseREST{
		elkeidFS: elkeidFS,
	}
}

type reverseREST struct {
	elkeidFS fs.FS
}

func (rest *reverseREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/reverse/elkeid").GET(rest.Elkeid)
}

func (rest *reverseREST) Elkeid(c *ship.Context) error {
	var req param.ReverseElkeid
	if err := c.BindQuery(&req); err != nil {
		return err
	}

	version, uname, arch := req.Version, req.Uname, req.Arch
	if arch == "" {
		arch = "amd64"
	}

	name := "hids_driver_" + version + "_" + uname + "_" + arch + ".ko"
	file, err := rest.elkeidFS.Open(name)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer file.Close()

	if stat, _ := file.Stat(); stat != nil {
		c.SetRespHeader(ship.HeaderContentLength, strconv.FormatInt(stat.Size(), 10))
	}
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": name})
	c.SetRespHeader(ship.HeaderContentDisposition, disposition)

	return c.Stream(http.StatusOK, ship.MIMEOctetStream, file)
}
