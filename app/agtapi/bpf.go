package agtapi

import (
	"math"
	"net/http"

	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/xgfone/ship/v5"
	"golang.org/x/net/bpf"
)

func BPF() route.Router {
	return &bpfREST{}
}

type bpfREST struct{}

func (rest *bpfREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/bpf/compile").POST(rest.Compile)
}

func (rest *bpfREST) Compile(c *ship.Context) error {
	var req param.BPFCompile
	if err := c.Bind(&req); err != nil {
		return err
	}

	expr := req.Expr
	c.Debugf("编译 bpf：%s", expr)

	link, length := layers.LinkTypeEthernet, math.MaxUint16
	fts, err := pcap.CompileBPFFilter(link, length, expr)
	if err != nil {
		return err
	}
	ins := make([]*bpf.RawInstruction, 0, len(fts))
	for _, ft := range fts {
		in := &bpf.RawInstruction{
			Op: ft.Code, Jt: ft.Jt, Jf: ft.Jf, K: ft.K,
		}
		ins = append(ins, in)
	}
	res := &param.Data{Data: ins}

	return c.JSON(http.StatusOK, res)
}
