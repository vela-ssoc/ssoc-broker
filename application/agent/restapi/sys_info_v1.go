package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/application/agent/request"
	"github.com/vela-ssoc/ssoc-broker/application/agent/service"
	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/xgfone/ship/v5"
)

type SysInfoV1 struct {
	svc *service.SysInfoV1
}

func NewSysInfoV1(svc *service.SysInfoV1) *SysInfoV1 {
	return &SysInfoV1{
		svc: svc,
	}
}

func (si *SysInfoV1) RegisterRoute(rgb *ship.RouteGroupBuilder) error {
	rgb.Route("/broker/collect/agent/sysinfo").POST(si.report)

	return nil
}

func (si *SysInfoV1) report(c *ship.Context) error {
	req := new(request.SysInfoV1Report)
	if err := c.Bind(req); err != nil {
		return err
	}
	info := req.Model()

	ctx := c.Request().Context()
	peer := muxserver.FromContext(ctx)

	return si.svc.Report(ctx, info, peer)
}
