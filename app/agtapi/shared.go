package agtapi

import (
	"net/http"

	"github.com/vela-ssoc/vela-broker/app/agtsvc"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/xgfone/ship/v5"
)

func Shared(stringsSvc agtsvc.SharedStringsService) route.Router {
	return &sharedAPI{
		stringsSvc: stringsSvc,
	}
}

type sharedAPI struct {
	stringsSvc agtsvc.SharedStringsService
}

func (api *sharedAPI) Route(r *ship.RouteGroupBuilder) {
	r.Route("/shared/strings/get").POST(api.StringsGet)
	r.Route("/shared/strings/set").POST(api.StringsSet)
	r.Route("/shared/strings/del").POST(api.StringsDel)
	r.Route("/shared/strings/incr").POST(api.StringsIncr)
}

func (api *sharedAPI) StringsGet(c *ship.Context) error {
	data := make([]*param.SharedKey, 0, 10)
	req := &param.Data{Data: &data}
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	dats, err := api.stringsSvc.Get(ctx, data)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &param.Data{Data: dats})
}

func (api *sharedAPI) StringsSet(c *ship.Context) error {
	data := make([]*param.SharedKeyValue, 0, 10)
	req := &param.Data{Data: &data}
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	return api.stringsSvc.Set(ctx, data, inf)
}

func (api *sharedAPI) StringsDel(c *ship.Context) error {
	data := make([]*param.SharedKey, 0, 10)
	req := &param.Data{Data: &data}
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return api.stringsSvc.Del(ctx, data)
}

func (api *sharedAPI) StringsIncr(c *ship.Context) error {
	data := make([]*param.SharedKeyIncr, 0, 10)
	req := &param.Data{Data: &data}
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dats, err := api.stringsSvc.Incr(ctx, data, inf)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &param.Data{Data: dats})
}
