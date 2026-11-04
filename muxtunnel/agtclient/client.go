package agtclient

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type Client struct {
	cli *http.Client
}

func NewClient(cli *http.Client) *Client {
	return &Client{
		cli: cli,
	}
}

func (c *Client) PushStartup(ctx context.Context, host string, data model.StartupV1) error {
	const path = "/api/v1/agent/startup"
	return c.postJSON(ctx, host, path, data, struct{}{})
}

func (c *Client) postJSON(ctx context.Context, host, path string, data, result any) error {
	body := new(bytes.Buffer)
	if err := json.NewEncoder(body).Encode(data); err != nil {
		return err
	}

	dest := muxproto.BuildAgentURL(host, path)
	target := dest.String()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, body)
	if err != nil {
		return err
	}
	res, err := c.send(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(result)
}

func (c *Client) send(req *http.Request) (*http.Response, error) {
	res, err := c.cli.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}
