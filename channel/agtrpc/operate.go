package agtrpc

import (
	"context"
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/channel/agtrpc/arequest"
	"github.com/vela-ssoc/ssoc-common/httpkit"
	"github.com/vela-ssoc/ssoc-common/linkhub"
)

type agentOperate struct {
	agentID int64
	cli     httpkit.Client
}

func (ac *agentOperate) Command(ctx context.Context, cmd string) error {
	const path = "/api/v1/agent/notice/command"
	reqURL := linkhub.NewBrokerToAgentIDURL(ac.agentID, path)
	req := &arequest.Command{Cmd: cmd}

	return ac.cli.SendJSON(ctx, http.MethodPost, reqURL.String(), nil, req, nil)
}
