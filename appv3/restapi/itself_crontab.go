package restapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/appv3/request"
	"github.com/vela-ssoc/ssoc-broker/appv3/response"
	"github.com/vela-ssoc/ssoc-broker/appv3/shipx"
	"github.com/vela-ssoc/ssoc-common-mb/crontab"
	"github.com/xgfone/ship/v5"
)

type ItselfCrontab struct {
	crond *crontab.Table
}

func NewItselfCrontab(crond *crontab.Table) *ItselfCrontab {
	return &ItselfCrontab{crond: crond}
}

func (slf *ItselfCrontab) RegisterHTTP(g shipx.RouteGroup) error {
	g.ManagerV3.Group.Route("/itself/crontab/jobs").GET(slf.jobs)
	g.ManagerV3.Group.Route("/itself/crontab/job").DELETE(slf.remove)

	return nil
}

func (slf *ItselfCrontab) jobs(c *ship.Context) error {
	infos := slf.crond.All()
	for i, info := range infos {
		if info.Sched != nil {
			infos[i].Expr = crontab.Stringify(info.Sched)
		}
	}
	ret := response.NewRecords(infos)

	return c.JSON(http.StatusOK, ret)
}

func (slf *ItselfCrontab) remove(c *ship.Context) error {
	var param request.StringID
	if err := c.BindQuery(&param); err != nil {
		return err
	}
	_, deleted := slf.crond.Remove(param.ID)
	ret := response.NewData(deleted)

	return c.JSON(http.StatusOK, ret)
}
