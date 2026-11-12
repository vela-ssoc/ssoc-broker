package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/response"
	"github.com/vela-ssoc/ssoc-broker/appv3/service"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"

	"github.com/xgfone/ship/v5"
)

type AgentStream struct {
	svc *service.AgentStream
}

func NewAgentStream(svc *service.AgentStream) *AgentStream {
	return &AgentStream{svc: svc}
}

func (bs *AgentStream) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/agent/stream").DELETE(bs.close)       // 结束主连接
	g.ManagerV3.Group.Route("/agent/substreams").GET(bs.substreams) // 获取子流信息
	g.ManagerV3.Group.Route("/agent/substream").DELETE(bs.kill)     // 强制结束一个子流

	return nil
}

func (bs *AgentStream) substreams(c *ship.Context) error {
	var param request.Int64ID
	if err := c.BindQuery(&param); err != nil {
		return err
	}

	subs, _ := bs.svc.Substreams(param.ID)

	var dats []response.AgentSubstream
	for _, sub := range subs {
		ss := sub.Stats()
		rs := response.AgentSubstream{
			ID:            ss.ID,
			RX:            ss.TX, // Broker 视角下，RX/TX 互换
			TX:            ss.RX, // Broker 视角下，RX/TX 互换
			EstablishedAt: ss.EstablishedAt,
		}
		dats = append(dats, rs)
	}
	ret := response.NewRecords(dats)

	return c.JSON(http.StatusOK, ret)
}

func (bs *AgentStream) kill(c *ship.Context) error {
	var param request.AgentSubstreamKill
	if err := c.BindQuery(&param); err != nil {
		return err
	}

	var err error
	subs, _ := bs.svc.Substreams(param.ID)
	for _, sub := range subs {
		ss := sub.Stats()
		if ss.ID == param.StreamID {
			err = sub.Close()
			break
		}
	}

	return err
}

func (bs *AgentStream) close(c *ship.Context) error {
	var param request.Int64ID
	if err := c.BindQuery(&param); err != nil {
		return err
	}
	ctx := c.Request().Context()

	return bs.svc.Close(ctx, param.ID)
}
