package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/vela-ssoc/ssoc-common-mb/problem"
	"golang.org/x/time/rate"
)

type Joiner interface {
	Name() string
	Auth(context.Context, Ident) (Issue, http.Header, bool, error)
	Join(context.Context, net.Conn, Ident, Issue) error
}

func New(joiner Joiner) http.Handler {
	maxsize := 150
	throughput := rate.NewLimiter(rate.Limit(maxsize), maxsize)

	return &minionGateway{
		name:       joiner.Name(),
		joiner:     joiner,
		throughput: throughput,
	}
}

type minionGateway struct {
	name   string
	joiner Joiner
	// throughput 限流器，防止 broker 上下线引起的
	// agent 节点蜂涌重连，拖慢数据库。
	throughput *rate.Limiter
}

func (gate *minionGateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 验证 HTTP 方法
	if method := r.Method; method != http.MethodConnect {
		gate.writeError(w, r, http.StatusBadRequest, "不支持的请求方法：%s", method)
		return
	}

	if !gate.throughput.Allow() {
		gate.writeError(w, r, http.StatusTooManyRequests, "请求过多稍候再试。")
		return
	}

	buf := make([]byte, 100*1024)
	n, _ := io.ReadFull(r.Body, buf)
	var ident Ident
	if err := ident.Decrypt(buf[:n]); err != nil {
		gate.writeError(w, r, http.StatusBadRequest, "认证信息错误")
		return
	}

	// 鉴权
	ctx := r.Context()
	issue, header, forbid, exx := gate.joiner.Auth(ctx, ident)
	if exx != nil {
		code := http.StatusBadRequest
		if forbid {
			code = http.StatusNotAcceptable
		}
		gate.writeError(w, r, code, "认证失败：%s", exx.Error())
		return
	}

	dat, err := issue.Encrypt()
	if err != nil {
		gate.writeError(w, r, http.StatusInternalServerError, "内部错误：%s", err.Error())
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		gate.writeError(w, r, http.StatusBadRequest, "协议错误")
		return
	}
	conn, _, jex := hijacker.Hijack()
	if jex != nil {
		gate.writeError(w, r, http.StatusBadRequest, "协议升级失败：%s", jex.Error())
		return
	}

	// -----[ Hijack Successful ]-----

	// 默认规定 http.StatusAccepted 为成功状态码
	code := http.StatusAccepted
	res := &http.Response{
		Status:     http.StatusText(code),
		StatusCode: code,
		Proto:      r.Proto,
		ProtoMajor: r.ProtoMajor,
		ProtoMinor: r.ProtoMinor,
		Header:     header,
		Request:    r,
	}
	if dsz := len(dat); dsz > 0 {
		res.Body = io.NopCloser(bytes.NewReader(dat))
		res.ContentLength = int64(dsz)
	}
	if err = res.Write(conn); err != nil {
		_ = conn.Close()
		return
	}

	if err = gate.joiner.Join(ctx, conn, ident, issue); err != nil {
		_ = conn.Close()
	}
}

// writeError 写入错误
func (gate *minionGateway) writeError(w http.ResponseWriter, r *http.Request, code int, msg string, args ...string) {
	if len(args) != 0 {
		msg = fmt.Sprintf(msg, args)
	}
	pd := &problem.Detail{
		Type:     gate.name,
		Title:    "节点接入验证不通过",
		Status:   code,
		Detail:   msg,
		Instance: r.RequestURI,
	}
	_ = pd.JSON(w)
}
