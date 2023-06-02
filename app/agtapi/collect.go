package agtapi

import (
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/xgfone/ship/v5"
)

func Collect() route.Router {
	return &collectREST{}
}

type collectREST struct{}

func (rest *collectREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/collect/agent/sysinfo").POST(rest.Sysinfo)
	r.Route("/broker/collect/agent/process").POST(rest.Process)
	r.Route("/broker/collect/agent/logon").POST(rest.Logon)
	r.Route("/broker/collect/agent/listen").POST(rest.Listen)
	r.Route("/broker/collect/agent/account").POST(rest.Account)
	r.Route("/broker/collect/agent/group").POST(rest.Group)
	r.Route("/broker/collect/agent/sbom").POST(rest.Sbom)
	r.Route("/broker/collect/agent/cpu").POST(rest.CPU)
}

func (rest *collectREST) Sysinfo(c *ship.Context) error {
	var req param.InfoRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dat := req.Model(inf.Issue().ID)
	err := query.SysInfo.WithContext(ctx).Save(dat)

	return err
}

func (rest *collectREST) Process(c *ship.Context) error {
	var req param.InfoRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dat := req.Model(inf.Issue().ID)
	err := query.SysInfo.WithContext(ctx).Save(dat)

	return err
}

func (rest *collectREST) Logon(c *ship.Context) error {
	var req param.InfoRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dat := req.Model(inf.Issue().ID)
	err := query.SysInfo.WithContext(ctx).Save(dat)

	return err
}

func (rest *collectREST) Listen(c *ship.Context) error {
	var req param.InfoRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dat := req.Model(inf.Issue().ID)
	err := query.SysInfo.WithContext(ctx).Save(dat)

	return err
}

func (rest *collectREST) Account(c *ship.Context) error {
	var req param.InfoRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dat := req.Model(inf.Issue().ID)
	err := query.SysInfo.WithContext(ctx).Save(dat)

	return err
}

func (rest *collectREST) Group(c *ship.Context) error {
	var req param.InfoRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dat := req.Model(inf.Issue().ID)
	err := query.SysInfo.WithContext(ctx).Save(dat)

	return err
}

func (rest *collectREST) Sbom(c *ship.Context) error {
	var req param.InfoRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dat := req.Model(inf.Issue().ID)
	err := query.SysInfo.WithContext(ctx).Save(dat)

	return err
}

func (rest *collectREST) CPU(c *ship.Context) error {
	var req param.CollectCPU
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	c.Infof("收到 %s CPU 上报信息", inf.Inet())

	return nil
}
