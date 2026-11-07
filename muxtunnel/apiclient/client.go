package apiclient

import (
	"log/slog"
	"net/http"

	"github.com/vela-ssoc/ssoc-common-mb/httpclient"
)

type Client struct {
	cli httpclient.Client
	log *slog.Logger
}

func NewClient(cli *http.Client, log *slog.Logger) Client {
	return Client{
		cli: httpclient.NewClient(cli),
		log: nil,
	}
}

func (c Client) Agent(host string) AgentClient {
	return AgentClient{
		cli:  c.cli,
		log:  c.log,
		host: host,
	}
}

func (c Client) Manager() ManagerClient {
	return ManagerClient{
		cli: c.cli,
		log: c.log,
	}
}
