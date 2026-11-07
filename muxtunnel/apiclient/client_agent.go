package apiclient

import (
	"log/slog"
	"net/url"

	"github.com/vela-ssoc/ssoc-common-mb/httpclient"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type AgentClient struct {
	cli  httpclient.Client
	log  *slog.Logger
	host string
}

func (ac AgentClient) Startup() AgentStartupClient {
	return AgentStartupClient{
		cli: ac.cli,
		log: ac.log,
		agt: ac,
	}
}

func (ac AgentClient) Task() AgentTaskClient {
	return AgentTaskClient{
		cli: ac.cli,
		log: ac.log,
		agt: ac,
	}
}

func (ac AgentClient) buildURL(path string) *url.URL {
	return muxproto.BuildAgentURL(ac.host, path)
}

func (ac AgentClient) buildStringURL(path string) string {
	return ac.buildURL(path).String()
}
