package serverd

import "github.com/vela-ssoc/ssoc-common/linkhub"

type AgentNotifier interface {
	AgentConnected(peer linkhub.Peer)
	AgentDisconnected(agentID int64)
}

type agentNotifier struct{}

func (agentNotifier) AgentConnected(linkhub.Peer) {}

func (agentNotifier) AgentDisconnected(int64) {}
