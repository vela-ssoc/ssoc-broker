package shipx

import "github.com/xgfone/ship/v5"

type RouteBinder interface {
	BindRoute(r *ship.RouteGroupBuilder) error
}

func BindRouters(r *ship.RouteGroupBuilder, rbs []RouteBinder) error {
	for _, rb := range rbs {
		if err := rb.BindRoute(r); err != nil {
			return err
		}
	}

	return nil
}

// RouteRegister 路由注册。
type RouteRegister interface {
	RegisterRoute(*ship.RouteGroupBuilder) error
}

// RegisterRoutes 将多个控制器注册至 base 路由。
func RegisterRoutes(base *ship.RouteGroupBuilder, routes []RouteRegister) error {
	for _, route := range routes {
		if err := route.RegisterRoute(base); err != nil {
			return err
		}
	}

	return nil
}
