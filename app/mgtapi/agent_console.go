package mgtapi

import (
	"strconv"
	"time"

	"github.com/vela-ssoc/ssoc-broker/library/pipelog"
	"github.com/vela-ssoc/ssoc-common/eventsource"
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
	r.Route("/agent/console/read").GET(ac.read)
	r.Route("/console/remove").POST(ac.remove)
}

func (ac *AgentConsole) read(c *ship.Context) error {
	req := new(agentConsoleRead)
	if err := c.BindQuery(req); err != nil {
		return err
	}

	w, r := c.Response(), c.Request()
	sse := eventsource.Accept(w, r)
	if sse == nil {
		return ship.ErrUnsupportedMediaType
	}

	name := strconv.FormatInt(req.ID, 10)
	ctx := r.Context()
	f, err := ac.fs.Open(name)
	if err != nil {
		return err
	}

	stat := agentConsoleStat{Maxsize: f.Maxsize()}
	if fi, _ := f.Stat(); fi != nil {
		stat.Size = fi.Size()
	}
	_ = sse.JSON("stat", stat)

	f.Subscriber(sse, req.N)
	defer f.Unsubscriber(sse)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var over bool
	for !over {
		select {
		case <-ctx.Done():
			over = true
		case <-ticker.C:
			cur := agentConsoleStat{Maxsize: f.Maxsize()}
			if fi, _ := f.Stat(); fi != nil {
				cur.Size = fi.Size()
			}

			if cur != stat {
				stat = cur
				_ = sse.JSON("stat", cur)
			}
		}
	}

	return nil
}

func (ac *AgentConsole) remove(c *ship.Context) error {
	req := new(agentConsoleRemove)
	if err := c.Bind(req); err != nil {
		return err
	}
	name := strconv.FormatInt(req.ID, 10)

	return ac.fs.Remove(name)
}

type agentConsoleRead struct {
	ID int64 `json:"id" form:"id" query:"id" validate:"required"`
	N  int   `json:"n"  form:"n"  query:"n"  validate:"required"`
}

type agentConsoleStat struct {
	Size    int64 `json:"size"`
	Maxsize int64 `json:"maxsize"`
}

type agentConsoleRemove struct {
	ID int64 `json:"id" form:"id" query:"id" validate:"required"`
}
