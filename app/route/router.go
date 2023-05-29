package route

import "github.com/xgfone/ship/v5"

type Router interface {
	Route(*ship.RouteGroupBuilder)
}
