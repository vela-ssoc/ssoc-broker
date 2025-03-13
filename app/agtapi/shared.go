package agtapi

import (
	"net/http"

	"github.com/vela-ssoc/ssoc-broker/app/agtsvc"
	"github.com/vela-ssoc/ssoc-broker/app/internal/param"
	"github.com/vela-ssoc/ssoc-broker/app/route"
	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
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
	req := new(param.SharedKey)
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	ret, err := api.stringsSvc.Get(ctx, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, ret)
}

func (api *sharedAPI) StringsSet(c *ship.Context) error {
	req := new(param.SharedKeyValue)
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	ret, err := api.stringsSvc.Set(ctx, inf, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, ret)
}

func (api *sharedAPI) StringsStore(c *ship.Context) error {
	req := new(param.SharedKeyValue)
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	ret, err := api.stringsSvc.Store(ctx, inf, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, ret)
}

func (api *sharedAPI) StringsIncr(c *ship.Context) error {
	req := new(param.SharedKeyIncr)
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	ret, err := api.stringsSvc.Incr(ctx, inf, req)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, ret)
}

func (api *sharedAPI) StringsDel(c *ship.Context) error {
	req := new(param.SharedKey)
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()

	return api.stringsSvc.Del(ctx, req)
}
