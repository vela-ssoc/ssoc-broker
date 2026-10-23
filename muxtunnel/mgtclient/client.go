package mgtclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type Client struct {
	cli *http.Client
}

func NewClient(cli *http.Client) Client {
	return Client{cli: cli}
}

func (c *Client) Ping(ctx context.Context) error {
	res, err := c.doGet(ctx, "/api/v3/broker/ping")
	if res != nil {
		_ = res.Body.Close()
	}

	return err
}

func (c *Client) doGet(ctx context.Context, path string) (*http.Response, error) {
	addr := c.buildURL(path)
	strURL := addr.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strURL, nil)
	if err != nil {
		return nil, err
	}

	return c.cli.Do(req)
}

func (c *Client) buildURL(path string) *url.URL {
	return &url.URL{Scheme: "http", Host: muxproto.ManagerFQDN, Path: path}
}
