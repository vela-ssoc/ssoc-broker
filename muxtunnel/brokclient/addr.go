package brokclient

import "net/url"

func NewEndpoint(hosts ...string) []*url.URL {
	addrs := make([]*url.URL, 0, len(hosts))
	for _, host := range hosts {
		u := &url.URL{Scheme: "wss", Host: host, Path: "/api/v3/tunnel"}
		addrs = append(addrs, u)
	}

	return addrs
}

func NewEndpoints(hosts ...string) []string {
	addrs := make([]string, 0, len(hosts))
	for _, u := range NewEndpoint(hosts...) {
		addrs = append(addrs, u.String())
	}

	return addrs
}
