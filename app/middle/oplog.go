package middle

import (
	"time"

	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/xgfone/ship/v5"
)

func Oplog(h ship.Handler) ship.Handler {
	return func(c *ship.Context) error {
		r := c.Request()
		method, reqURL := r.Method, r.URL.String()
		desc, ok := c.Route.Data.(route.Describer)

		sat := time.Now()
		err := h(c)
		du := time.Since(sat)
		if ok && desc.Ignore(du) && err == nil { // 无需记录
			return err
		}

		var name string
		if ok {
			name = desc.Name()
		}

		if err == nil {
			c.Infof("[%12s] %s %s %s", du, name, method, reqURL)
			return nil
		}
		c.Warnf("[%12s] %s %s %s %s", du, name, method, reqURL, err)

		return err
	}
}
