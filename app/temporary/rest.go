package temporary

import (
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xgfone/ship/v5"
)

type httpREST struct {
	upgrade *websocket.Upgrader
	log     *slog.Logger
	valid   ship.Validator
	handler Handler
}

func REST(handler Handler, valid ship.Validator, log *slog.Logger) *httpREST {
	upgrade := &websocket.Upgrader{
		HandshakeTimeout:  10 * time.Second,
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: true,
	}

	return &httpREST{
		upgrade: upgrade,
		log:     log,
		valid:   valid,
		handler: handler,
	}
}

//func (rest *httpREST) Route(sh *ship.Ship) {
//	sh.Route("/api/minion/endpoint").GET(rest.Endpoint)
//	sh.Route("/api/edition/upgrade").GET(rest.Upgrade)
//}

// Endpoint 接入点
//
//	400 http.StatusBadRequest      : 认证错误
//	425 http.StatusTooEarly        : 尚未激活
//	429 http.StatusTooManyRequests : 重复登录
func (rest *httpREST) Endpoint(c *ship.Context) error {
	// 解密参数
	var ident Ident
	if err := ident.decrypt(c.GetReqHeader(ship.HeaderAuthorization)); err != nil {
		rest.log.Warn("minion 认证信息解密失败", slog.Any("error", err))
		return c.NoContent(http.StatusBadRequest)
	}
	// 校验参数
	if err := rest.valid.Validate(ident); err != nil {
		rest.log.Info("minion 认证信息校验失败", slog.Any("error", err))
		return c.NoContent(http.StatusBadRequest)
	}

	inet := ident.Inet.String()
	// 认证授权
	claim, err := rest.handler.Authorize(ident)
	if err != nil {
		rest.log.Info("minion 授权认证失败", slog.Any("error", err), slog.String("inet", inet))
		return rest.authErr(err)
	}
	// 校验授权信息
	if err = rest.valid.Validate(claim); err != nil {
		rest.log.Info("minion 授权信息校验失败", slog.Any("error", err), slog.String("inet", inet))
		return c.NoContent(http.StatusBadRequest)
	}
	// 加密认证信息并设置到响应 header
	enc, err := claim.encrypt()
	if err != nil {
		rest.log.Warn("minion 授权信息加密错误", slog.Any("error", err), slog.String("inet", inet))
		return c.NoContent(http.StatusBadRequest)
	}
	header := http.Header{ship.HeaderAuthorization: []string{enc}}
	ws, err := rest.upgrade.Upgrade(c.ResponseWriter(), c.Request(), header)
	if err != nil {
		rest.log.Warn("minion websocket upgrade 错误", slog.Any("error", err), slog.String("inet", inet))
		return rest.authErr(err)
	}

	conn := rest.newConn(ws, ident, claim)
	defer func() {
		_ = conn.close()
		rest.handler.Disconnect(conn)
	}()
	rest.handler.Connect(conn)
	id := claim.ID
	rest.log.Info("minion 建立连接", slog.Int64("minion_id", id), slog.String("inet", inet))

	// 读取 broker 发来的消息
	timeout := rest.handler.Timeout()
	for {
		rec, ex := conn.receive(timeout)
		if ex != nil {
			if rest.closedErr(ex) {
				rest.log.Warn("minion 断开连接", slog.Any("error", ex), slog.String("inet", inet), slog.Int64("minion_id", id))
				break
			}
			rest.log.Warn("minion 消息读取发生临时错误", slog.Any("error", ex), slog.String("inet", inet), slog.Int64("minion_id", id))
			continue
		}
		rest.handler.Receive(conn, rec)
	}

	return nil
}

func (rest *httpREST) newConn(ws *websocket.Conn, ident Ident, claim Claim) *Conn {
	return &Conn{conn: ws, ident: ident, claim: claim, validator: rest.valid}
}

// authErr 错误转化
func (*httpREST) authErr(err error) error {
	switch ev := err.(type) {
	case nil, *ship.HTTPServerError:
		return ev
	default:
		return ship.ErrBadRequest.New(err)
	}
}

// closedErr 判断错误是否是连接关闭错误
func (*httpREST) closedErr(err error) bool {
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}

	switch err.(type) {
	case *websocket.CloseError, *net.OpError, net.Error:
		return true
	default:
		return false
	}
}
