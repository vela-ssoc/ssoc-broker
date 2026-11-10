package muxacct

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
	velasmux "github.com/vela-ssoc/vela-common-mba/smux"
	"github.com/vela-ssoc/vela-tunnel/muxconn"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func (srv *Server) AcceptTCP(w http.ResponseWriter, r *http.Request) error {
	sess := &sessionDataV1{ConnectedAt: time.Now()}
	node, err := srv.authenticationV1(w, r, sess)
	if err != nil {
		return err
	}

	srv.opts.Event.OnConnected(node)
	err = srv.serveHTTP(node)
	sess.DisconnectedAt = time.Now()
	srv.disconnectedV1(node, sess)
	srv.opts.Event.OnDisconnected(node)

	return err
}

func (srv *Server) authenticationV1(w http.ResponseWriter, r *http.Request, sess *sessionDataV1) (muxproto.Node, error) {
	payload := make([]byte, 100*1024) // 100K 一般是足够存放认证报文了。
	n, err := io.ReadFull(r.Body, payload)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		srv.log.Error("读取认证报文错误", "err", err)
		return nil, err
	}

	hijacker, support := w.(http.Hijacker)
	if !support {
		srv.log.Error("客户端连接不支持 Hijacker")
		return nil, err
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		srv.log.Warn("客户端连接 Hijack 出错", "err", err)
		return nil, http.ErrHijacked
	}

	var ident IdentV1
	if err = ident.Decrypt(payload[:n]); err != nil {
		srv.log.Error("认证报文解析错误", "err", err)
		return nil, err
	}
	if err = srv.validateIdentV1(ident); err != nil {
		srv.log.Error("认证报文验证出错", "err", err)
		_ = srv.writeErrorV1(conn, r, http.StatusBadRequest, err)
		return nil, err
	}
	sess.Ident = ident

	// TODO 限流器检查
	minion, err := srv.findOrCreateByInetV1(ident, sess.ConnectedAt)
	if err != nil {
		_ = srv.writeErrorV1(conn, r, http.StatusBadRequest, err)
		return nil, err
	}
	if minion.Status == model.MSOnline {
		err = errors.New("节点已经在线了（数据库检查）")
		_ = srv.writeErrorV1(conn, r, http.StatusConflict, err)
		srv.log.Error("节点重复上线", "ident", ident, "err", err)
		return nil, err
	} else if minion.Status == model.MSDelete {
		err = errors.New("节点已被标记删除")
		_ = srv.writeErrorV1(conn, r, http.StatusForbidden, err)
		srv.log.Error("节点已被标记删除", "ident", ident, "err", err)
		return nil, err
	}
	sess.Minion = minion

	velacfg := velasmux.DefaultConfig()
	if ident.Keepalive {
		velacfg.KeepAliveDisabled = false
		srv.log.Info("此 vela-tunnel 开启了 keepalive", "ident", ident)
	}

	mux, err := muxconn.NewVela(context.Background(), conn, velacfg, true)
	if err != nil {
		_ = srv.writeErrorV1(conn, r, http.StatusBadRequest, err)
		srv.log.Error("升级到多路通道出错", "ident", ident, "err", err)
		return nil, err
	}

	info := muxproto.NodeInfo{
		IDV1:    minion.ID,
		Inet:    ident.Inet,
		GOOS:    ident.GOOS,
		GOARCH:  ident.Arch,
		Version: ident.Semver,
	}
	host := strconv.Itoa(int(minion.ID))
	node := srv.hub.Put(host, mux, info)
	if node == nil {
		err = errors.New("节点已经在线了（连接池检查）")
		_ = srv.writeErrorV1(conn, r, http.StatusBadRequest, err)
		srv.log.Error("节点重复上线", "ident", ident, "err", err)
		return nil, err
	}

	// 修改数据库中的上线状态
	if err = srv.updateOnlineV1(minion, ident, sess.ConnectedAt); err != nil {
		srv.hub.Del(host)
		_ = srv.writeErrorV1(conn, r, http.StatusBadRequest, err)
		return nil, err
	}

	// 回写成功报文
	iss := IssueV1{ID: minion.ID}
	err = srv.writeSuccessV1(conn, r, iss)
	if err == nil {
		// 刷新标签状态
		_ = srv.updateTagsV1(minion, ident)

		srv.log.Info("节点上线验证成功", "ident", ident)
		return node, nil
	}

	// 撤回下线状态
	if ret, err1 := srv.rollbackOfflineV1(minion.ID); err1 != nil {
		srv.log.Error("回滚节点上线状态出错", "ident", ident, "err", err)
	} else if ret.RowsAffected == 0 {
		srv.log.Error("回滚节点上线状态异常（无任何修改）", "ident", ident)
	}
	srv.hub.Del(host) // 连接池回滚

	return nil, err
}

func (srv *Server) findOrCreateByInetV1(ident IdentV1, now time.Time) (*model.Minion, error) {
	ctx, cancel := srv.perContext()
	defer cancel()

	inet, machineID := ident.Inet, ident.MachineID
	tbl := srv.pdb.RDB().Minion
	dao := tbl.WithContext(ctx)

	var conds []gen.Condition
	if machineID == "" {
		conds = []gen.Condition{tbl.Inet.Eq(inet), tbl.MachineID.Eq("")}
	} else {
		conds = []gen.Condition{tbl.MachineID.Eq(machineID)}
	}

	dat, err := dao.Where(conds...).First()
	if err == nil {
		return dat, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		srv.log.Error("查询探针节点错误", "ident", ident, "err", err)
		return nil, err
	}

	doc := &model.Minion{
		MachineID:   ident.MachineID,
		Inet:        ident.Inet,
		Status:      model.MSOffline,
		HeartbeatAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err = dao.Create(doc); err != nil {
		srv.log.Error("数据库新建探针节点错误", "ident", ident, "err", err)
		return nil, err
	}
	srv.log.Info("添加注册了新的探针节点", "ident", ident)

	return doc, nil
}

func (srv *Server) updateOnlineV1(mon *model.Minion, ident IdentV1, now time.Time) error {
	ctx, cancel := srv.perContext()
	defer cancel()

	tbl := srv.pdb.RDB().Minion
	dao := tbl.WithContext(ctx)

	bcfg := srv.mux.Config()
	info, err := dao.Where(tbl.ID.Eq(mon.ID), tbl.Status.Eq(uint8(model.MSOffline))).
		UpdateSimple(
			tbl.Status.Value(uint8(model.MSOnline)),
			tbl.Inet.Value(ident.Inet),
			tbl.MAC.Value(ident.MAC),
			tbl.Goos.Value(ident.GOOS),
			tbl.Arch.Value(ident.Arch),
			tbl.Edition.Value(ident.Semver),
			tbl.Unstable.Value(ident.Unstable),
			tbl.Customized.Value(ident.Customized),
			tbl.Uptime.Value(sql.NullTime{Time: now, Valid: true}),
			tbl.BrokerID.Value(bcfg.IDV1),
			tbl.BrokerName.Value(bcfg.Name),
			tbl.BrokerSlug.Value(bcfg.Slug),
		)
	if err != nil {
		srv.log.Error("修改节点上线失败", "ident", ident, "err", err)
		return err
	} else if info.RowsAffected == 0 {
		err = errors.New("修改节点上线失败（没有修改任何数据）")
		srv.log.Error("修改节点上线失败", "ident", ident, "err", err)
		return err
	}

	return nil
}

func (srv *Server) updateTagsV1(minion *model.Minion, ident IdentV1) error {
	ctx, cancel := srv.perContext()
	defer cancel()

	qry := srv.pdb.RDB()
	id, inet := minion.ID, minion.Inet

	return qry.Transaction(func(tx *query.Query) error {
		const kind = model.TkLifelong
		tags := model.MinionTags{
			&model.MinionTag{MinionID: id, Tag: inet, Kind: kind},
		}
		if goos := ident.GOOS; goos != "" {
			tags = append(tags, &model.MinionTag{MinionID: id, Tag: goos, Kind: kind})
		}
		if arch := ident.Arch; arch != "" {
			tags = append(tags, &model.MinionTag{MinionID: id, Tag: arch, Kind: kind})
		}

		minionTagTbl := tx.MinionTag
		dao := minionTagTbl.WithContext(ctx)
		// 1. 删除所有的内置标签
		_, _ = dao.Where(minionTagTbl.MinionID.Eq(id), minionTagTbl.Kind.Eq(int8(kind))).Delete()
		// 2. 插入新的内置标签
		_ = dao.Create(tags...)

		return nil
	})
}

func (srv *Server) disconnectedV1(node muxproto.Node, sess *sessionDataV1) {
	host, mux, ident := node.Host(), node.MUX(), sess.Ident
	defer func() {
		_ = mux.Close()
		srv.hub.Del(host)
	}()

	srv.log.Info("探针节点下线了", "ident", ident)

	ctx, cancel := srv.perContext()
	defer cancel()

	tbl := srv.pdb.RDB().Minion
	dao := tbl.WithContext(ctx)

	mid := sess.Minion.ID
	ret, err := dao.Where(tbl.ID.Eq(mid), tbl.Status.Eq(uint8(model.MSOnline))).UpdateSimple(
		tbl.Status.Value(uint8(model.MSOffline)),
	)
	if err != nil {
		srv.log.Info("修改节点下线状态出错", "ident", ident, "err", err)
	} else if ret.RowsAffected == 0 {
		srv.log.Info("修改节点下线状态出错（没有修改任何数据）", "ident", ident)
	}
}

func (srv *Server) rollbackOfflineV1(id int64) (gen.ResultInfo, error) {
	ctx, cancel := srv.perContext()
	defer cancel()

	tbl := srv.pdb.RDB().Minion
	dao := tbl.WithContext(ctx)

	return dao.Where(tbl.ID.Eq(id), tbl.Status.Eq(uint8(model.MSOnline))).
		UpdateSimple(tbl.Status.Value(uint8(model.MSOffline)))
}

func (srv *Server) validateIdentV1(ident IdentV1) error {
	if mid := ident.MachineID; mid == "" {
		// v4 及以上版本必须含有机器码
		majors, _ := strings.CutSuffix(ident.Semver, ".")
		if major, _ := strconv.Atoi(majors); major >= 4 {
			srv.log.Warn("高版本探针程序必须携带机器码", "ident", ident)
			return fmt.Errorf("此版本 %s 必须携带机器码", majors)
		}
	}

	if fn := srv.opts.Validate; fn != nil {
		return fn(ident)
	}

	return nil
}

func (srv *Server) perContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), srv.perTimeout())
}

func (srv *Server) perTimeout() time.Duration {
	if d := srv.opts.PerTimeout; d > 0 {
		return d
	}

	return 30 * time.Second
}

func (srv *Server) serveHTTP(node muxproto.Node) error {
	h := srv.next
	if h == nil {
		h = http.NotFoundHandler()
	}
	hs := &http.Server{
		Handler: h,
		BaseContext: func(net.Listener) context.Context {
			return muxproto.WithContext(context.Background(), node)
		},
	}

	mux := node.MUX()
	//goland:noinspection GoUnhandledErrorResult
	defer mux.Close()

	return hs.Serve(mux)
}

// writeErrorV1 旧版 agent 上线失败响应消息。
//
//goland:noinspection GoUnhandledErrorResult
func (srv *Server) writeErrorV1(conn net.Conn, r *http.Request, status int, err error) error {
	defer conn.Close()

	body := new(bytes.Buffer)
	auth := &muxproto.AuthError{Code: status, Text: err.Error()}
	if err = json.NewEncoder(body).Encode(auth); err != nil {
		return err
	}

	return srv.writeBodyV1(conn, r, status, body)
}

//goland:noinspection GoUnhandledErrorResult
func (srv *Server) writeBodyV1(conn net.Conn, r *http.Request, status int, body *bytes.Buffer) error {
	res := &http.Response{
		Status:        http.StatusText(status),
		StatusCode:    status,
		Proto:         r.Proto,
		ProtoMajor:    r.ProtoMajor,
		ProtoMinor:    r.ProtoMinor,
		Request:       r,
		Body:          io.NopCloser(body),
		ContentLength: int64(body.Len()),
	}

	d := srv.perTimeout()
	conn.SetWriteDeadline(time.Now().Add(d))
	err := res.Write(conn)
	conn.SetWriteDeadline(time.Time{}) // 设置 deadline 后一定要清除。

	return err
}

func (srv *Server) writeSuccessV1(conn net.Conn, r *http.Request, iss IssueV1) error {
	enc, err := iss.Encrypt()
	if err != nil {
		return err
	}

	return srv.writeBodyV1(conn, r, http.StatusAccepted, bytes.NewBuffer(enc))
}
