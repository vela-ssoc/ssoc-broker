package agtapi

import (
	"net/http"
	"net/url"

	"github.com/vela-ssoc/vela-broker/app/agtsvc"
	"github.com/vela-ssoc/vela-broker/app/internal/modview"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/xgfone/ship/v5"
)

func Deploy(svc agtsvc.DeployService) *deployREST {
	return &deployREST{
		svc: svc,
	}
}

type deployREST struct {
	svc agtsvc.DeployService
}

func (rest *deployREST) Script(c *ship.Context) error {
	var req param.DeployMinionDownload
	if err := c.BindQuery(&req); err != nil {
		return err
	}

	r := c.Request()
	reqURL := r.URL

	scheme := "http"
	if c.IsTLS() {
		scheme = "https"
	}
	// 如果 TLS 证书挂在了 WAF 上
	proto := c.GetReqHeader(ship.HeaderXForwardedProto)
	if proto == "http" || proto == "https" {
		scheme = proto
	}

	path := reqURL.Path + "/download"
	downURL := &url.URL{
		Scheme:   scheme,
		Host:     r.Host,
		Path:     path,
		RawQuery: reqURL.RawQuery,
	}
	ctx := c.Request().Context()

	data := &modview.Deploy{DownloadURL: downURL}
	read, err := rest.svc.Script(ctx, req.Goos, data)
	if err == nil {
		return c.Stream(http.StatusOK, ship.MIMETextPlainCharsetUTF8, read)
	}

	redirectURL := &url.URL{
		Path:     path,
		RawQuery: r.URL.RawQuery,
	}

	return c.Redirect(http.StatusTemporaryRedirect, redirectURL.String())
}

func (rest *deployREST) MinionDownload(c *ship.Context) error {
	var req param.DeployMinionDownload
	if err := c.BindQuery(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	file, err := rest.svc.OpenMinion(ctx, &req)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer file.Close()

	// 此时的 Content-Length = 原始文件 + 隐藏文件
	c.Header().Set(ship.HeaderContentLength, file.ContentLength())
	c.Header().Set(ship.HeaderContentDisposition, file.Disposition())

	return c.Stream(http.StatusOK, file.ContentType(), file)
}
