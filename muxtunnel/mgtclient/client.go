package mgtclient

import (
	"context"
	"net/http"

	"github.com/vela-ssoc/ssoc-proto/muxproto"
	"github.com/vela-ssoc/ssoc-proto/muxtool"
)

type Client struct {
	base muxtool.Client
}

func NewClient(base muxtool.Client) Client {
	return Client{base: base}
}

func (c Client) Base() muxtool.Client {
	return c.base
}

func (c Client) Ping(ctx context.Context) error {
	reqURL := muxproto.ToManagerURL("/api/v1/heartbeat")
	strURL := reqURL.String()

	return c.base.JSON(ctx, http.MethodGet, strURL, nil)
}
