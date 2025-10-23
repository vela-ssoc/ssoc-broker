package shipx

import "github.com/xgfone/ship/v5"

type RouteBinder interface {
	BindRoute(rgb *ship.RouteGroupBuilder) error
}

func BindRoutes(base *ship.RouteGroupBuilder, rbs []RouteBinder) error {
	for _, rb := range rbs {
		if rb == nil {
			continue
		}

		if err := rb.BindRoute(base); err != nil {
			return err
		}
	}

	return nil
}
