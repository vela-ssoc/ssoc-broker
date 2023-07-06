package temporary

import "fmt"

type Mode string

const (
	ModeKafka  Mode = "kafka"
	ModeTunnel      = "tunnel"
	ModeHTTP        = "http"
	ModeES          = "es"
)

var modes = map[Mode]string{
	ModeKafka:  "kafka 代理通道",
	ModeTunnel: "socket 代理通道",
	ModeHTTP:   "http 代理通道",
	ModeES:     "es 代理通道",
}

// String implement fmt.Stringer
func (m Mode) String() string {
	if str, exist := modes[m]; exist {
		return str
	}
	return fmt.Sprintf("<unnamed minion stream mode: %s>", string(m))
}
