package temporary

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/xgfone/ship/v5"
)

type Conn struct {
	conn      *websocket.Conn
	ident     Ident
	claim     Claim
	validator ship.Validator
	wmu       sync.Mutex
}

func (c *Conn) Ident() Ident {
	return c.ident
}

func (c *Conn) Claim() Claim {
	return c.claim
}

func (c *Conn) ID() int64 {
	return c.claim.ID
}

func (c *Conn) Inet() net.IP {
	return c.ident.Inet
}

// Authed 判断 token 是否一致
func (c *Conn) Authed(token string) bool {
	return c.claim.Token == token
}

func (c *Conn) Send(msg *Message) error {
	if msg == nil {
		return nil
	}
	if c.conn == nil {
		return io.ErrUnexpectedEOF
	}
	msg.mask = c.claim.Mask
	raw, err := msg.marshal()
	if err != nil {
		return err
	}

	// 发送消息要加锁, 遇到并发调用接收端可能会导致数据错乱
	c.wmu.Lock()
	defer c.wmu.Unlock()

	return c.conn.WriteMessage(websocket.BinaryMessage, raw)
}

// receive 读取消息
func (c *Conn) receive(timeout time.Duration) (*Receive, error) {
	if c.conn == nil {
		return nil, io.EOF
	}

	_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
	_, raw, err := c.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	rec := &Receive{minionID: c.claim.ID, mask: c.claim.Mask, validator: c.validator}
	err = rec.unmarshal(raw)

	return rec, err
}

// close 关闭连接
func (c *Conn) close() error {
	if conn := c.conn; conn != nil {
		return conn.Close()
	}
	return nil
}
