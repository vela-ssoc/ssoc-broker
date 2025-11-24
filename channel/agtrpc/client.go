package agtrpc

import "github.com/vela-ssoc/ssoc-common/httpkit"

type Client interface {
	Operator(agentID int64) Operator
}

func NewClient(cli httpkit.Client) Client {
	return &agentClient{cli: cli}
}

type agentClient struct {
	cli httpkit.Client
}

func (ac *agentClient) Operator(agentID int64) Operator {
	return &agentOperate{
		agentID: agentID,
		cli:     ac.cli,
	}
}
