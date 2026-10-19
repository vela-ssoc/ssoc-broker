package shipx

import (
	"net/http"
	"strings"

	"github.com/xgfone/ship/v5"
)

func Strip(c *ship.Context, key string) (http.ResponseWriter, *http.Request) {
	w, r := c.Response(), c.Request()

	rpath := r.URL.Path   // raw path
	upath := c.Param(key) // upstream path

	if strings.HasSuffix(rpath, "/") { // 如果原始路径尾部带 / 就补上。
		upath = upath + "/"
	}
	if strings.HasPrefix(rpath, "/") {
		upath = "/" + upath
	}
	r.URL.Path = upath

	query := r.URL.Query()
	query.Del("authorization")
	r.URL.RawQuery = query.Encode()
	r.Header.Del(ship.HeaderAuthorization)

	return w, r
}
