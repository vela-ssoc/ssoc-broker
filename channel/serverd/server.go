package serverd

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mba/smux"
	"gorm.io/gorm"
)

type Handler interface {
	Handle(sess *smux.Session)
}

func New(qry *query.Query) {
	return
}

type agentServer struct {
	qry *query.Query
	opt option
}

func (as *agentServer) Handle(sess *smux.Session) {
	// 开始读取握手信息
}

func (as *agentServer) authentication(sess *smux.Session, timeout time.Duration) error {
	timer := time.AfterFunc(timeout, func() { _ = sess.Close() })
	sig, err := sess.AcceptStream()
	timer.Stop()
	if err != nil {
		return err
	}
	defer sig.Close()

	_ = sig.SetDeadline(time.Now().Add(timeout))
	req, err := as.readRequest(sig)
	if err != nil {
		return err
	}
	if valid := as.opt.valid; valid != nil {
		if err = valid(req); err != nil {
			return err
		}
	}

	if err = as.join(req, timeout); err != nil {
		return err
	}

	return nil
}

func (as *agentServer) join(req *authRequest, timeout time.Duration) error {
	mon, err := as.findOrCreate(req, timeout)
	if err != nil {
		return err
	}
	// 检查状态是否允许上线
	switch mon.Status {
	case model.MSOnline: // 已经在线的不能上线
	case model.MSDelete: // 标记为已删除的不允许上线
	}

	return nil
}

func (as *agentServer) readRequest(stm *smux.Stream) (*authRequest, error) {
	head := make([]byte, 4)
	if n, err := io.ReadFull(stm, head); err != nil {
		return nil, err
	} else if n != 4 {
		return nil, io.ErrShortBuffer
	}

	size := binary.BigEndian.Uint32(head)
	data := make([]byte, size)
	if n, err := io.ReadFull(stm, data); err != nil {
		return nil, err
	} else if n != int(size) {
		return nil, io.ErrShortBuffer
	}

	req := new(authRequest)
	err := json.Unmarshal(data, req)

	return req, err
}

func (as *agentServer) writeResponse(stm *smux.Stream, resp *authResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	size := len(data)
	if size > 65535 {
		return io.ErrShortBuffer
	}

	head := make([]byte, 4)
	binary.BigEndian.PutUint32(head, uint32(size))

	if _, err = stm.Write(head); err == nil {
		_, err = stm.Write(data)
	}

	return err
}

func (as *agentServer) findOrCreate(req *authRequest, timeout time.Duration) (*model.Minion, error) {
	machineID := req.MachineID

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	tbl := as.qry.Minion
	dao := tbl.WithContext(ctx)
	if data, err := dao.Where(tbl.MachineID.Eq(machineID)).First(); err == nil {
		return data, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// 自动创建 agent 节点
	data := &model.Minion{
		MachineID:  machineID,
		Inet:       req.Inet,
		Goos:       req.Goos,
		Arch:       req.Goarch,
		Edition:    req.Semver,
		Status:     model.MSOffline,
		Unload:     req.Unload,
		Unstable:   req.Unstable,
		Customized: req.Customized,
	}
	if err := dao.Create(data); err != nil {
		return nil, err
	}

	return data, nil
}
