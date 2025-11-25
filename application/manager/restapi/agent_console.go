package restapi

import (
	"strconv"

	"github.com/vela-ssoc/ssoc-broker/application/manager/request"
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

func (ac *AgentConsole) BindRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/agent/console/watch").GET(ac.watch)
	rgb.Route("/agent/console/remove").GET(ac.remove)
	return nil
}

func (ac *AgentConsole) watch(c *ship.Context) error {
	req := new(request.Int64ID)
	if err := c.BindQuery(req); err != nil {
		return err
	}

	w, r := c.Response(), c.Request()
	sse := eventsource.Accept(w, r)
	if sse == nil {
		return ship.ErrUnsupportedMediaType
	}

	ctx := r.Context()
	name := strconv.FormatInt(req.ID, 10)
	f, err := ac.fs.Open(name)
	if err != nil {
		return err
	}
	f.Subscriber(sse, 10)
	defer f.Unsubscriber(sse)

	select {
	case <-ctx.Done():
	}

	return nil
}

func (ac *AgentConsole) remove(c *ship.Context) error {
	req := new(request.Int64ID)
	if err := c.BindQuery(req); err != nil {
		return err
	}
	name := strconv.FormatInt(req.ID, 10)

	return ac.fs.Remove(name)
}
