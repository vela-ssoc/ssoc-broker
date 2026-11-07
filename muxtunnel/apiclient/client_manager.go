package apiclient

import (
	"log/slog"
	"net/url"

	"github.com/vela-ssoc/ssoc-common-mb/httpclient"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type ManagerClient struct {
	cli httpclient.Client
	log *slog.Logger
}

func (mc ManagerClient) buildURL(path string) *url.URL {
	return muxproto.BuildManagerURL(path)
}

func (mc ManagerClient) buildStringURL(path string) string {
	return mc.buildURL(path).String()
}
