package agtapi

import (
	"net/http"
	"time"

	"github.com/vela-ssoc/ssoc-broker/app/internal/param"
	"github.com/vela-ssoc/ssoc-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/xgfone/ship/v5"
)

func Security(qry *query.Query) route.Router {
	return &securityREST{
		qry:   qry,
		limit: 1000,
	}
}

type securityREST struct {
	qry   *query.Query
	limit int
}

func (rest *securityREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/ip/risk").POST(rest.RiskIP)
	r.Route("/broker/ip/pass").POST(rest.PassIP)
	r.Route("/broker/dns/risk").POST(rest.RiskDNS)
	r.Route("/broker/dns/pass").POST(rest.PassDNS)
	r.Route("/broker/file/risk").POST(rest.PassDNS)
}

func (rest *securityREST) RiskIP(c *ship.Context) error {
	var qry param.SecurityKindRequest
	if err := c.BindQuery(&qry); err != nil {
		return err
	}
	var body param.SecurityIPRequest
	if err := c.Bind(&body); err != nil {
		return err
	}

	now := time.Now()
	ctx := c.Request().Context()
	tbl := rest.qry.RiskIP
	dao := tbl.WithContext(ctx).
		Where(tbl.IP.In(body.Data...), tbl.BeforeAt.Gte(now))
	if len(qry.Kind) != 0 {
		dao.Where(tbl.Kind.In(qry.Kind...))
	}
	dats, _ := dao.Limit(rest.limit).Find()
	kinds := model.RiskIPs(dats).IPKinds()
	res := &param.SecurityResult{
		Count: len(kinds),
		Data:  kinds,
	}

	return c.JSON(http.StatusOK, res)
}

func (rest *securityREST) PassIP(c *ship.Context) error {
	var qry param.SecurityKindRequest
	if err := c.BindQuery(&qry); err != nil {
		return err
	}
	var body param.SecurityIPRequest
	if err := c.Bind(&body); err != nil {
		return err
	}

	now := time.Now()
	ctx := c.Request().Context()
	tbl := rest.qry.PassIP
	dao := tbl.WithContext(ctx).
		Where(tbl.IP.In(body.Data...), tbl.BeforeAt.Gte(now))
	if len(qry.Kind) != 0 {
		dao.Where(tbl.Kind.In(qry.Kind...))
	}
	dats, _ := dao.Limit(rest.limit).Find()
	kinds := model.PassIPs(dats).IPKinds()
	res := &param.SecurityResult{
		Count: len(kinds),
		Data:  kinds,
	}

	return c.JSON(http.StatusOK, res)
}

func (rest *securityREST) RiskDNS(c *ship.Context) error {
	var qry param.SecurityKindRequest
	if err := c.BindQuery(&qry); err != nil {
		return err
	}
	var body param.SecurityDNSRequest
	if err := c.Bind(&body); err != nil {
		return err
	}

	now := time.Now()
	ctx := c.Request().Context()
	tbl := rest.qry.RiskDNS
	dao := tbl.WithContext(ctx).
		Where(tbl.Domain.In(body.Data...), tbl.BeforeAt.Gte(now))
	if len(qry.Kind) != 0 {
		dao.Where(tbl.Kind.In(qry.Kind...))
	}
	dats, _ := dao.Limit(rest.limit).Find()
	kinds := model.RiskDNSs(dats).DomainKinds()
	res := &param.SecurityResult{
		Count: len(kinds),
		Data:  kinds,
	}

	return c.JSON(http.StatusOK, res)
}

func (rest *securityREST) PassDNS(c *ship.Context) error {
	var qry param.SecurityKindRequest
	if err := c.BindQuery(&qry); err != nil {
		return err
	}
	var body param.SecurityDNSRequest
	if err := c.Bind(&body); err != nil {
		return err
	}

	now := time.Now()
	ctx := c.Request().Context()
	tbl := rest.qry.PassDNS
	dao := tbl.WithContext(ctx).
		Where(tbl.Domain.In(body.Data...), tbl.BeforeAt.Gte(now))
	if len(qry.Kind) != 0 {
		dao.Where(tbl.Kind.In(qry.Kind...))
	}
	dats, _ := dao.Limit(rest.limit).Find()
	kinds := model.PassDNSs(dats).DomainKinds()
	res := &param.SecurityResult{
		Count: len(kinds),
		Data:  kinds,
	}

	return c.JSON(http.StatusOK, res)
}

func (rest *securityREST) RiskFile(c *ship.Context) error {
	var qry param.SecurityKindRequest
	if err := c.BindQuery(&qry); err != nil {
		return err
	}
	var body param.SecurityFileRequest
	if err := c.Bind(&body); err != nil {
		return err
	}

	ctx := c.Request().Context()
	tbl := rest.qry.RiskFile
	dao := tbl.WithContext(ctx).
		Where(tbl.Checksum.In(body.Data...), tbl.BeforeAt.Gte(time.Now()))
	if len(qry.Kind) != 0 {
		dao.Where(tbl.Kind.In(qry.Kind...))
	}

	// Limit 做下限制防止数据量太大
	dats, err := dao.Limit(rest.limit).Find()
	if err != nil {
		return err
	}
	hashmap := model.RiskFiles(dats).ChecksumKinds()
	res := &param.SecurityResult{Data: hashmap, Count: len(hashmap)}

	return c.JSON(http.StatusOK, res)
}
