package apiclient

import (
	"context"
	"log/slog"

	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/httpclient"
)

type AgentStartupClient struct {
	cli httpclient.Client
	log *slog.Logger
	agt AgentClient
}

func (asc AgentStartupClient) Push(ctx context.Context, in model.StartupV1) error {
	strURL := asc.agt.buildStringURL("/api/v1/agent/startup")
	out := new(EmptyOut)

	return asc.cli.PostJSON(ctx, strURL, in, out)
}
