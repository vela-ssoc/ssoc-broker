package middle

import "github.com/xgfone/ship/v5"

func MustWebsocket(h ship.Handler) ship.Handler {
	return func(c *ship.Context) error {
		if !c.IsWebSocket() {
			return ship.ErrBadRequest
		}

		return h(c)
	}
}
