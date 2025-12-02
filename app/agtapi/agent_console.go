package agtapi

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
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

	var from string // 配置脚本名
	if fv, exists := data["from"]; exists {
		from, _ = fv.(string)
	}
	sum := sha1.Sum([]byte(from)) // 将文件名 hash 一下，防止文件包含敏感字符文件系统冲突
	fullname := name + "-" + hex.EncodeToString(sum[:])

	f, err := ac.fs.Open(fullname)
	if err != nil {
		return err
	}
	_, _ = f.Write(msg)

	return nil
}
