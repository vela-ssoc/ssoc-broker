package telecom

import (
	"net"
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

type Addresses []*Address

// Preformat 对地址进行格式化处理，即：如果地址内有显式端口号，
// 则根据是否开启 TLS 补充默认端口号
func (ads Addresses) Preformat() {
	for _, ad := range ads {
		addr := ad.Addr
		host, port, err := net.SplitHostPort(addr)
		if err == nil && port != "" {
			if ad.Name == "" {
				ad.Name = host
			}
			continue
		}
		if ad.Name == "" {
			ad.Name = addr
		}
		if ad.TLS {
			ad.Addr = addr + ":443"
		} else {
			ad.Addr = addr + ":80"
		}
	}
}
