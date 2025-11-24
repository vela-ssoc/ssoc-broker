package agtapi

import (
	"bytes"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
	"github.com/vela-ssoc/ssoc-broker/library/pipelog"
	"github.com/xgfone/ship/v5"
)

func NewAgentConsole(fs pipelog.FS) *AgentConsole {
	return &AgentConsole{
		fs: fs,
	}
}

type AgentConsole struct {
	fs pipelog.FS
}

func (ac *AgentConsole) Route(r *ship.RouteGroupBuilder) {
	r.Route("/console/write").POST(ac.write)
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
	ctx := r.Context()
	inf := mlink.Ctx(ctx)
	id := inf.Issue().ID
	inet := inf.Inet().String()
	name := strconv.FormatInt(id, 10)
	data["id"] = id
	data["inet"] = inet
	data["time"] = now
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}
	if !bytes.HasSuffix(msg, []byte("\n")) {
		msg = append(msg, '\n')
	}
	f, err := ac.fs.Open(name)
	if err != nil {
		return err
	}
	_, _ = f.Write(msg)

	return nil
}
