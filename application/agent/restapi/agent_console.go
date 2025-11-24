package restapi

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/vela-ssoc/ssoc-broker/library/pipelog"
	"github.com/vela-ssoc/ssoc-common/linkhub"
	"github.com/xgfone/ship/v5"
)

type AgentConsole struct {
	fs pipelog.FS
}

func NewAgentConsole(fs pipelog.FS) *AgentConsole {
	return &AgentConsole{
		fs: fs,
	}
}

func (ac *AgentConsole) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/console/write").POST(ac.write)
	return nil
}

func (ac *AgentConsole) write(c *ship.Context) error {
	r := c.Request()
	rd := io.LimitReader(r.Body, 1024*1024)
	body, err := io.ReadAll(rd)
	if err != nil {
		return err
	}

	data := make(map[string]any, 8)
	if err = json.Unmarshal(body, &data); err != nil {
		data["raw_body"] = body
	}

	now := time.Now()
	ctx := c.Request().Context()
	peer := linkhub.FromContext(ctx)
	info := peer.Info()
	data["id"] = info.ID
	data["inet"] = info.Inet
	data["time"] = now
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if !bytes.HasSuffix(msg, []byte("\n")) {
		msg = append(msg, '\n')
	}
	f, err := ac.fs.Open(info.Host)
	if err != nil {
		return err
	}
	_, _ = f.Write(msg)

	return nil
}
