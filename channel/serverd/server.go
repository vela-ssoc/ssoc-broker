package serverd

import (
	"context"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common-mb/options"
	"github.com/vela-ssoc/ssoc-common/linkhub"
	"github.com/xtaci/smux"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type Handler interface {
	Handle(sess *smux.Session)
}

func New(qry *query.Query, cur *model.Broker, opts ...options.Lister[option]) Handler {
	opts = append(opts, fallbackOption())
	opt := options.Eval(opts...)

	return &agentServer{
		qry: qry,
		cur: cur,
		opt: opt,
	}
}

type agentServer struct {
	qry *query.Query
	cur *model.Broker
	opt option
}

func (as *agentServer) Handle(sess *smux.Session) {
	defer sess.Close()

	// 如果 broker 重启，会导致 agent 同时下线，那么重启上线时间也基本一致，
	// 为了防止 agent 蜂拥上线对 broker 造成压力，可以通过限流器分批上线。
	if !as.opt.limit.Allowed() {
		as.log().Error("限流器阻止 agent 建立连接")
		return
	}

	timeout := as.opt.timeout
	peer, _, err := as.authentication(sess, timeout)
	if err != nil {
		as.log().Warn("节点上线认证失败", "error", err)
		return
	}
	defer as.disconnect(peer, timeout)

	as.opt.notifier.AgentConnected(peer)

	srv := as.opt.server
	base := srv.BaseContext
	srv.BaseContext = func(ln net.Listener) context.Context {
		parent := context.Background()
		if base != nil {
			parent = base(ln)
		}
		return linkhub.WithContext(parent, peer)
	}

	lis := &smuxListener{sess: sess}
	err = srv.Serve(lis)

	as.log().Warn("agent 节点下线了", "error", err)
}

func (as *agentServer) authentication(sess *smux.Session, timeout time.Duration) (linkhub.Peer, *model.Minion, error) {
	timer := time.AfterFunc(timeout, func() { _ = sess.Close() })
	sig, err := sess.AcceptStream()
	timer.Stop()
	if err != nil {
		return nil, nil, err
	}
	defer sig.Close()

	_ = sig.SetDeadline(time.Now().Add(timeout))
	req, err := as.readRequest(sig)
	if err != nil {
		return nil, nil, err
	}
	if err = as.opt.valid(req); err != nil {
		return nil, nil, err
	}

	mon, peer, code, err1 := as.join(sess, req, timeout)
	err2 := as.writeResponse(sig, code, err1)
	if err1 != nil {
		return nil, mon, err1
	} else if err2 != nil {
		return nil, mon, err2
	}

	return peer, mon, nil
}

func (as *agentServer) join(sess *smux.Session, req *authRequest, timeout time.Duration) (*model.Minion, linkhub.Peer, int, error) {
	attrs := []any{slog.Any("agent_auth_request", req), slog.Duration("timeout", timeout)}
	mon, err := as.findOrCreate(req, timeout)
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		as.log().Error("查找或自动新增 agent 节点发生错误", attrs...)
		return nil, nil, http.StatusInternalServerError, err
	}
	// 检查状态是否允许上线
	status := mon.Status
	switch status {
	case model.MSOnline: // 已经在线的不能上线
		as.log().Warn("agent 节点已经在线了（数据库检查）", attrs...)
		return mon, nil, http.StatusConflict, errors.New("节点重复上线")
	case model.MSDelete: // 标记为已删除的不允许上线
		as.log().Info("agent 节点已标记为删除", attrs...)
		return mon, nil, http.StatusForbidden, errors.New("节点已标记为删除")
	default:
		if status != model.MSOffline && status != model.MSInactive {
			attrs = append(attrs, slog.Any("status", status))
			as.log().Info("节点数据库状态异常", attrs...)
			return mon, nil, http.StatusForbidden, errors.New("节点数据库状态异常")
		}
	}

	minionID := mon.ID
	peer := linkhub.NewPeer(minionID, req.Inet, sess)
	if !as.opt.huber.Put(peer) {
		as.log().Warn("agent 节点已经在线了（内存检查）", attrs...)
		return mon, nil, http.StatusConflict, errors.New("节点重复上线")
	}

	// 修改数据库中的在线状态
	offline, online := uint8(model.MSOffline), uint8(model.MSOnline)
	tbl := as.qry.Minion
	updates := []field.AssignExpr{
		tbl.Status.Value(online),
		tbl.Inet.Value(req.Inet),
		tbl.Goos.Value(req.Goos),
		tbl.Arch.Value(req.Goarch),
		tbl.Edition.Value(req.Semver),
		tbl.Uptime.Value(sql.NullTime{Time: time.Now(), Valid: true}),
		tbl.Unload.Value(req.Unload),
		tbl.Unstable.Value(req.Unstable),
		tbl.Customized.Value(req.Customized),
		tbl.BrokerID.Value(as.cur.ID),
		tbl.BrokerName.Value(as.cur.Name),
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if _, err = tbl.WithContext(ctx).
		Where(tbl.ID.Eq(minionID), tbl.Status.Eq(offline)).
		UpdateSimple(updates...); err != nil {
		as.opt.huber.DelByID(minionID)
		attrs = append(attrs, slog.Any("error", err))
		as.log().Warn("将数据库的节点状态标记为在线发生成错误", attrs...)
		return mon, nil, http.StatusInternalServerError, err
	}

	// 修改持久化标签
	tags := []*model.MinionTag{
		{MinionID: minionID, Tag: req.Inet, Kind: model.TkLifelong},
		{MinionID: minionID, Tag: req.Goos, Kind: model.TkLifelong},
		{MinionID: minionID, Tag: req.Goarch, Kind: model.TkLifelong},
	}
	_ = as.qry.Transaction(func(tx *query.Query) error {
		ttbl := tx.MinionTag
		tdao := ttbl.WithContext(ctx)
		_, _ = tdao.Where(ttbl.MinionID.Eq(minionID), ttbl.Kind.Eq(int8(model.TkLifelong))).Delete()
		_ = tdao.Create(tags...)

		return nil
	})

	as.log().Info("agent 上线成功", attrs...)

	return mon, peer, http.StatusOK, nil
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

func (as *agentServer) writeResponse(stm *smux.Stream, code int, err error) error {
	resp := &authResponse{Code: code}
	if err != nil {
		resp.Message = err.Error()
	}
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

func (as *agentServer) disconnect(peer linkhub.Peer, timeout time.Duration) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	id := peer.Info().ID
	tbl := as.qry.Minion
	dao := tbl.WithContext(ctx)
	online, offline := uint8(model.MSOnline), uint8(model.MSOffline)
	if _, err := dao.Where(tbl.ID.Eq(id), tbl.Status.Eq(online), tbl.BrokerID.Eq(as.cur.ID)).
		UpdateSimple(tbl.Status.Value(offline)); err != nil {
		as.log().Warn("修改节点下线状态失败", "error", err)
	}
	as.opt.huber.DelByID(id)
	as.opt.notifier.AgentDisconnected(id)
}

func (as *agentServer) log() *slog.Logger {
	if l := as.opt.logger; l != nil {
		return l
	}

	return slog.Default()
}

type smuxListener struct {
	sess *smux.Session
}

func (sl *smuxListener) Accept() (net.Conn, error) {
	stm, err := sl.sess.AcceptStream()
	if err != nil {
		return nil, err
	}

	return stm, nil
}

func (sl *smuxListener) Close() error {
	return sl.sess.Close()
}

func (sl *smuxListener) Addr() net.Addr {
	return sl.sess.LocalAddr()
}
