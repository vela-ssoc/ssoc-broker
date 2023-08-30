package telecom

import (
	"net"
	"net/url"
	"strings"
)

// Address 连接地址
type Address struct {
	// TLS 服务端是否开了 TLS
	TLS bool `json:"tls"  yaml:"tls"`

	// Addr 服务端地址，只需要填写地址或地址端口号，不需要路径
	// Example:
	//  	- soc.xxx.com
	//  	- soc.xxx.com:9090
	//		- 10.10.10.2
	// 		- 10.10.10.2:9090
	// 如果没有显式指明端口号，则开启 TLS 默认为 443，未开启 TLS 默认为 80
	Addr string `json:"addr" yaml:"addr"`

	// Name 主机名或 TLS SNI 名称
	// 无论是否开启 TLS，在发起 HTTP 请求时该 Name 都会被设置为 Host。
	// 当开启 TLS 时该 Name 会被设置为校验证书的 Servername。
	// 如果该字段为空，则默认使用 Addr 的地址作为主机名。
	Name string `json:"name" yaml:"name"`
}

// String fmt.Stringer
func (ad Address) String() string {
	build := new(strings.Builder)
	build.WriteString("[")
	if ad.TLS {
		build.WriteString("tls://")
	} else {
		build.WriteString("tcp://")
	}
	build.WriteString(ad.Addr)

	build.WriteString(" host/servername: ")
	build.WriteString(ad.Name)
	build.WriteString("]")

	return build.String()
}

// Addresses broker 地址切片
type Addresses []*Address

// Preformat 对地址进行格式化处理，即：如果地址内有显式端口号，
// 则根据是否开启 TLS 补充默认端口号
func (ads Addresses) Preformat() Addresses {
	size := len(ads)
	tmap := make(map[string]*Address, size)
	smap := make(map[string]*Address, size)
	ret := make(Addresses, 0, 2*size)
	for _, ad := range ads {
		host, port := ads.splitHostPort(ad.Addr)
		sport, tport := port, port
		if port == "" {
			sport, tport = "443", "80"
		}
		name := ad.Name
		if name == "" {
			name = host
		}

		saddr := net.JoinHostPort(host, sport)
		if old := smap[saddr]; old == nil {
			addr := &Address{TLS: true, Addr: saddr, Name: name}
			smap[saddr] = addr
			ret = append(ret, addr)
		}

		taddr := net.JoinHostPort(host, tport)
		if old := tmap[taddr]; old == nil {
			addr := &Address{Addr: taddr, Name: name}
			tmap[saddr] = addr
			ret = append(ret, addr)
		}
	}

	return ret
}

// splitHostPort 截取 host 和 port
func (Addresses) splitHostPort(str string) (string, string) {
	if u, err := url.Parse(str); err == nil {
		if u.Host != "" && u.Scheme != "" {
			str = u.Host
		}
	}

	if host, port, err := net.SplitHostPort(str); err == nil {
		return host, port
	}

	return str, ""
}
