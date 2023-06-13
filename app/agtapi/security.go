package agtapi

import (
	"net/http"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/xgfone/ship/v5"
)

func Security() route.Router {
	return &securityREST{}
}

type securityREST struct{}

func (rest *securityREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/ip/risk").POST(rest.RiskIP)
	r.Route("/broker/ip/pass").POST(rest.PassIP)
	r.Route("/broker/dns/risk").POST(rest.RiskDNS)
	r.Route("/broker/dns/pass").POST(rest.PassDNS)
	r.Route("/broker/file/risk").POST(rest.PassDNS)
}

func (rest *securityREST) RiskIP(c *ship.Context) error {
	return nil
}

func (rest *securityREST) PassIP(c *ship.Context) error {
	return nil
}

func (rest *securityREST) RiskDNS(c *ship.Context) error {
	return nil
}

func (rest *securityREST) PassDNS(c *ship.Context) error {
	return nil
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
	tbl := query.RiskFile
	dao := tbl.WithContext(ctx).
		Where(tbl.Checksum.In(body.Data...), tbl.BeforeAt.Gte(time.Now()))
	if len(qry.Kind) != 0 {
		dao.Where(tbl.Kind.In(qry.Kind...))
	}

	// Limit 做下限制防止数据量太大
	dats, err := dao.Limit(1000).Find()
	if err != nil {
		return err
	}
	hashmap := model.RiskFiles(dats).ChecksumKinds()
	res := &param.SecurityFileResult{Data: hashmap, Count: len(hashmap)}

	return c.JSON(http.StatusOK, res)
}
