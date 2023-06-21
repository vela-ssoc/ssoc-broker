package agtapi

import (
	"mime"
	"net/http"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/app/service"
	"github.com/xgfone/ship/v5"
)

func Third(svc service.ThirdService) route.Router {
	return &thirdREST{
		svc: svc,
	}
}

type thirdREST struct {
	svc service.ThirdService
}

func (rest *thirdREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/third").Data(route.Named("agent 下载三方文件")).GET(rest.Download)
}

func (rest *thirdREST) Download(c *ship.Context) error {
	var req param.ThirdDownload
	if err := c.BindQuery(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	third, file, err := rest.svc.Open(ctx, req.Name)
	if err != nil {
		return ship.ErrNotFound
	}
	//goland:noinspection GoUnhandledErrorResult
	defer file.Close()
	if third.Hash == req.Hash {
		c.WriteHeader(http.StatusNotModified)
		return nil
	}

	params := map[string]string{
		"filename": third.Name,
		"hash":     third.Hash,
	}
	disposition := mime.FormatMediaType("attachment", params)
	c.Header().Set(ship.HeaderContentLength, file.ContentLength())
	c.Header().Set(ship.HeaderContentDisposition, disposition)

	return c.Stream(http.StatusOK, file.ContentType(), file)
}
