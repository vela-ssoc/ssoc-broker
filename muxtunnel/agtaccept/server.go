package agtaccept

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	mathrand "math/rand/v2"
	"net"
	"net/http"
	"time"

	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"github.com/vela-ssoc/ssoc-proto/muxconn"
	"github.com/vela-ssoc/ssoc-proto/muxproto"
	"github.com/vela-ssoc/ssoc-proto/muxtool"
	velasmux "github.com/vela-ssoc/vela-common-mba/smux"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Options struct {
	Huber      muxserver.Huber // 必须填写
	Handler    http.Handler
	Validator  func(any) error
	Logger     *slog.Logger
	ThisBroker func() (bson.ObjectID, string)
	PerTimeout time.Duration
	BootLoader muxserver.BootLoader[muxproto.AgentBootConfig] // 必须填写，节点认证通过后加载启动配置。
	Notifier   muxserver.ConnectNotifier
}

func NewAccept(db repository.Database, opts Options) muxserver.MUXAccepter {
	return &agentAccept{
		db:   db,
		opts: opts,
	}
}

type agentAccept struct {
	db   repository.Database
	opts Options
}

func (srv *agentAccept) AcceptMUX(mux muxconn.Muxer) error {
	return nil
}

// AcceptTCP 该接口用于兼容老版本 agent 节点上线。
//
// 老版本的多路复用连接方式有诸多问题，将会逐渐淘汰掉，新版将完全迁移至 [agentAccept.AcceptMUX]。
func (srv *agentAccept) AcceptTCP(w http.ResponseWriter, r *http.Request) error {
	connectAt := time.Now()
	sessData := &tunnelSessionDataV1{ConnectAt: connectAt}
	peer, err := srv.authenticationV1(w, r, sessData)
	if err != nil {
		srv.log().Warn("节点认证上线是失败", "session", sessData, "error", err)

		if ntf := srv.opts.Notifier; ntf == nil {
			srv.log().Debug("没有注册回调函数，无需触发认证失败回调")
		} else {
			srv.log().Debug("触发认证失败回调")

			ctx, cancel := srv.perContext()
			defer cancel()

			ntf.OnAuthFailed(ctx, nil, connectAt, err)
		}

		return err
	}

	srv.log().Debug("开始服务节点业务", "session", sessData)
	err = srv.serveHTTP(peer)
	sessData.DisconnectAt = time.Now()
	srv.log().Debug("节点下线", "session", sessData, "error", err)

	srv.disconnected(sessData)

	return nil
}

func (srv *agentAccept) authenticationV1(w http.ResponseWriter, r *http.Request, sessData *tunnelSessionDataV1) (muxserver.Peer, error) {
	buf := make([]byte, 100*1024) // 100K 一般是足够存放认证报文了。
	n, _ := io.ReadFull(r.Body, buf)
	req := new(IdentV1)
	if err := req.Decrypt(buf[:n]); err != nil {
		srv.log().Error("认证报文解析错误", "session", sessData, "error", err)
		return nil, err
	}
	sessData.Request = req

	hijacker, support := w.(http.Hijacker)
	if !support {
		srv.log().Warn("客户端连接不支持 hijacker", "session", sessData)
		return nil, errors.New("客户端连接不支持 hijacker")
	}
	conn, _, err := hijacker.Hijack()
	if err != nil {
		srv.log().Warn("客户端连接 hijack 出错", "session", sessData, "error", err)
		return nil, err
	}
	// broker 视角记录 agent 的地址，raddr laddr 互换。
	sessData.RemoteAddr = conn.LocalAddr().String()
	sessData.LocalAddr = conn.RemoteAddr().String()

	// 参数校验
	if pde := srv.validateV1(req); pde != nil {
		srv.writeErrorV1(conn, r, pde) // 响应错误信息。
		srv.log().Warn("认证报文校验错误", "session", sessData, "error", pde)
		return nil, errors.New(pde.Detail)
	}

	node, pde := srv.findOrCreateV1(req)
	if pde != nil {
		srv.writeErrorV1(conn, r, pde) // 响应错误信息。
		srv.log().Error("查询/创建节点错误", "session", sessData, "error", err)
		return nil, err
	}
	if node.Status == model.MinionStatusDelete {
		detail := "节点已被标记删除"
		pde = &muxtool.ProblemDetails{Status: http.StatusForbidden, Detail: detail}
		srv.writeErrorV1(conn, r, pde) // 响应错误信息。
		srv.log().Error(detail, "session", sessData)
		return nil, errors.New(detail)
	} else if node.Status == model.MinionStatusOnline {
		detail := "节点已经在线（数据库检查）"
		pde = &muxtool.ProblemDetails{Status: http.StatusConflict, Detail: detail}
		srv.writeErrorV1(conn, r, pde) // 响应错误信息。
		srv.log().Error(detail, "session", sessData)
		return nil, errors.New(detail)
	}

	info := muxserver.PeerInfo{
		Instance:    req.MachineID,
		Semver:      req.Semver,
		Inet:        req.Inet,
		Goos:        req.Goos,
		Goarch:      req.Arch,
		Hostname:    req.Hostname,
		ConnectedAt: sessData.ConnectAt,
	}
	if info.Instance == "" {
		info.Instance = info.Inet
	}

	var passwd []byte
	muxcfg := velasmux.DefaultConfig()
	muxcfg.KeepAliveDisabled = false
	if r.TLS == nil { // 如果未配置 TLS 加密，通道就开启加密传输。
		passwd = srv.generatePasswd()
		muxcfg.Passwd = passwd
	}
	mux := muxconn.NewVela(nil, conn, muxcfg, true)
	name, module := mux.Library()

	// 加入到连接池中
	id := node.ID
	sessData.ID = id
	sessData.TunnelLibrary = model.TunnelLibrary{Name: name, Module: module}
	peer, details := srv.putHub(id, mux, info)
	if details != nil {
		srv.log().Warn(details.Detail, "session", sessData)
		srv.writeErrorV1(conn, r, details)
		return nil, errors.New(details.Detail)
	}
	sessData.Peer = peer

	// 节点上线
	if pde = srv.updateOnlineV1(sessData, node); pde != nil {
		srv.delHub(id) // 从连接池中删除
		srv.log().Error(pde.Detail, "session", sessData)
		srv.writeErrorV1(conn, r, pde)
		return nil, errors.New(pde.Detail)
	}

	// 回写成功消息
	resp := &IssueV1{Passwd: passwd, SID: id.Hex()}
	if err = srv.writeSuccessV1(conn, r, resp); err != nil {
		srv.delHub(id) // 从连接池中删除
		srv.log().Error("写入响应报文出错", "session", sessData, "error", err)

		return nil, err
	}

	return peer, nil
}

func (srv *agentAccept) findOrCreateV1(req *IdentV1) (*model.Minion, *muxtool.ProblemDetails) {
	if req.MachineID != "" {
		return srv.findOrCreateByMachineIDV1(req)
	}

	return srv.findOrCreateByInetV1(req)
}

func (srv *agentAccept) findOrCreateByMachineIDV1(req *IdentV1) (*model.Minion, *muxtool.ProblemDetails) {
	ctx, cancel := srv.perContext()
	defer cancel()

	coll := srv.db.Minion()
	machineID := req.MachineID
	dat, err := coll.FindOne(ctx, bson.D{{Key: "machine_id", Value: machineID}})
	if err == nil {
		return dat, nil
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, &muxtool.ProblemDetails{Status: http.StatusInternalServerError, Detail: err.Error()}
	}

	return srv.findOrCreateByInetV1(req)
}

func (srv *agentAccept) findOrCreateByInetV1(req *IdentV1) (*model.Minion, *muxtool.ProblemDetails) {
	ctx, cancel := srv.perContext()
	defer cancel()

	inet := req.Inet
	coll := srv.db.Minion()
	filter := bson.M{"machine_id": "", "inet": inet}
	dat, err := coll.FindOne(ctx, filter)
	if err == nil {
		return dat, nil
	} else if !errors.Is(err, mongo.ErrNoDocuments) {
		srv.log().Error("通过 inet 查询节点错误", "inet", inet, "error", err)
		return nil, &muxtool.ProblemDetails{Status: http.StatusInternalServerError, Detail: err.Error()}
	}

	return srv.createMinionV1(req)
}

// createMinionV1 新增 agent 节点。
func (srv *agentAccept) createMinionV1(req *IdentV1) (*model.Minion, *muxtool.ProblemDetails) {
	now := time.Now()
	doc := &model.Minion{
		MachineID: req.MachineID,
		Inet:      req.Inet,
		Status:    model.MinionStatusOffline,
		CreatedAt: now,
		UpdatedAt: now,
	}

	coll := srv.db.Minion()
	ctx, cancel := srv.perContext()
	defer cancel()

	ret, err := coll.InsertOne(ctx, doc)
	if err != nil {
		srv.log().Error("节点新增错误", "new_minion", doc, "error", err)
		return nil, &muxtool.ProblemDetails{Status: http.StatusInternalServerError, Detail: err.Error()}
	}
	id, ok := ret.InsertedID.(bson.ObjectID)
	if !ok {
		msg := "新增 ID 类型错误"
		srv.log().Error(msg, "inserted_id", id)
		return nil, &muxtool.ProblemDetails{Status: http.StatusInternalServerError, Detail: msg}
	}
	doc.ID = id

	return doc, nil
}

func (srv *agentAccept) updateOnlineV1(sessData *tunnelSessionDataV1, node *model.Minion) *muxtool.ProblemDetails {
	// 每次上线都要更新标签
	req := sessData.Request
	goos, goarch, inet := req.Goos, req.Arch, req.Inet
	tags := node.Tags.ReplaceAllSystemTags(inet, goos, goarch)

	var broker model.MinionBroker
	if fn := srv.opts.ThisBroker; fn != nil {
		broker.ID, broker.Name = fn()
	}

	execStat := model.ExecuteStat{
		Goos:       goos,
		Goarch:     goarch,
		Semver:     req.Semver,
		Version:    model.Semver(req.Semver).Uint64(),
		Unstable:   req.Unstable,
		PID:        req.PID,
		Args:       []string{},
		Hostname:   req.Hostname,
		Workdir:    req.Workdir,
		Executable: req.Executable,
	}
	tunStat := model.TunnelStat{
		Inet:        inet,
		ConnectedAt: sessData.ConnectAt,
		KeepaliveAt: sessData.ConnectAt,
		Library:     sessData.TunnelLibrary,
		LocalAddr:   sessData.LocalAddr,
		RemoteAddr:  sessData.RemoteAddr,
	}
	sessData.TunnelStat = tunStat
	sessData.ExecuteStat = execStat
	sessData.Broker = broker

	filter := bson.M{"_id": sessData.ID, "status": model.MinionStatusOffline}
	update := bson.M{"$set": bson.M{
		"machine_id":   req.MachineID,
		"status":       model.MinionStatusOnline,
		"tags":         tags,
		"unload":       req.Unload,
		"broker":       broker,
		"tunnel_stat":  tunStat,
		"execute_stat": execStat,
	}}

	ctx, cancel := srv.perContext()
	defer cancel()

	coll := srv.db.Minion()
	if _, err := coll.UpdateOne(ctx, filter, update); err != nil {
		srv.log().Error("修改节点在线状态错误", "session", sessData)
		return &muxtool.ProblemDetails{Status: http.StatusInternalServerError, Detail: err.Error()}
	}

	return nil
}

func (srv *agentAccept) serveHTTP(peer muxserver.Peer) error {
	h := srv.opts.Handler
	if h == nil {
		h = http.NotFoundHandler()
	}
	hs := &http.Server{
		Handler: h,
		BaseContext: func(net.Listener) context.Context {
			return muxserver.WithContext(context.Background(), peer)
		},
	}
	mux := peer.MUX()

	return hs.Serve(mux)
}

// writeErrorV1 旧版 agent 上线失败响应消息。
//
//goland:noinspection GoUnhandledErrorResult
func (srv *agentAccept) writeErrorV1(conn net.Conn, r *http.Request, pde *muxtool.ProblemDetails) {
	defer conn.Close()

	body := new(bytes.Buffer)
	json.NewEncoder(body).Encode(pde)

	status := pde.Status
	srv.writeBodyV1(conn, r, status, body)
}

func (srv *agentAccept) writeSuccessV1(conn net.Conn, r *http.Request, resp *IssueV1) error {
	enc, err := resp.Encrypt()
	if err != nil {
		return err
	}

	return srv.writeBodyV1(conn, r, http.StatusAccepted, bytes.NewBuffer(enc))
}

//goland:noinspection GoUnhandledErrorResult
func (srv *agentAccept) writeBodyV1(conn net.Conn, r *http.Request, status int, body *bytes.Buffer) error {
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
	conn.SetWriteDeadline(time.Time{}) // 设置 deadline 后一定要清除 deadline

	return err
}

func (srv *agentAccept) validateV1(req *IdentV1) *muxtool.ProblemDetails {
	if v := srv.opts.Validator; v != nil {
		if err := v(req); err != nil {
			return &muxtool.ProblemDetails{
				Status: http.StatusBadRequest,
				Detail: err.Error(),
			}
		}

		return nil
	}
	// TODO 校验参数
	return nil
}

func (srv *agentAccept) log() *slog.Logger {
	if srv.opts.Logger != nil {
		return srv.opts.Logger
	}

	return slog.Default()
}

func (srv *agentAccept) perTimeout() time.Duration {
	if d := srv.opts.PerTimeout; d > 0 {
		return d
	}

	return 30 * time.Second
}

func (srv *agentAccept) perContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), srv.perTimeout())
}

// generatePasswd 随机生成一个 32-64 位长度的加密密钥。
func (*agentAccept) generatePasswd() []byte {
	size := mathrand.Int32N(33) + 32
	passwd := make([]byte, size)
	_, _ = rand.Read(passwd)

	return passwd
}

func (srv *agentAccept) putHub(id bson.ObjectID, mux muxconn.Muxer, info muxserver.PeerInfo) (muxserver.Peer, *muxtool.ProblemDetails) {
	if srv.opts.Huber == nil {
		return nil, &muxtool.ProblemDetails{
			Status: http.StatusNotImplemented,
			Detail: "内部错误：没有配置连接池",
		}
	}

	peer := srv.opts.Huber.Put(id, mux, info)
	if peer == nil {
		return nil, &muxtool.ProblemDetails{
			Status: http.StatusConflict,
			Detail: "节点已经在线（连接池检查）",
		}
	}

	return peer, nil
}

func (srv *agentAccept) delHub(id bson.ObjectID) {
	if hub := srv.opts.Huber; hub != nil {
		hub.DelID(id)
	}
}

func (srv *agentAccept) disconnected(sessData *tunnelSessionDataV1) {
	id, peer := sessData.ID, sessData.Peer
	tx, rx := peer.MUX().Traffic()
	{
		filter := bson.D{{Key: "_id", Value: id}, {Key: "status", Value: model.MinionStatusOnline}}
		update := bson.M{"$set": bson.M{
			"status":                      model.MinionStatusOffline,
			"tunnel_stat.disconnected_at": sessData.DisconnectAt,
			"tunnel_stat.receive_bytes":   rx,
			"tunnel_stat.transmit_bytes":  tx,
		}}

		ctx, cancel := srv.perContext()
		defer cancel()

		coll := srv.db.Minion()
		ret, err := coll.UpdateOne(ctx, filter, update)
		if err != nil {
			srv.log().Error("节点下线修改数据库出错", "session", sessData, "error", err)
		} else if ret.ModifiedCount <= 0 {
			srv.log().Error("节点下线修改数据库未匹配到数据", "session", sessData)
		} else {
			srv.log().Debug("节点下线修改数据库完毕", "session", sessData)
		}
	}
	srv.delHub(id) // 从 hub 中删除连接

	{
		tunStat := sessData.TunnelStat
		tunStatHis := model.TunnelStatHistory{
			Inet:           tunStat.Inet,
			ConnectedAt:    sessData.ConnectAt,
			DisconnectedAt: sessData.DisconnectAt,
			ConnectSeconds: sessData.connectedSeconds(),
			Library:        tunStat.Library,
			LocalAddr:      sessData.LocalAddr,
			RemoteAddr:     sessData.RemoteAddr,
			ReceiveBytes:   rx,
			TransmitBytes:  tx,
		}
		his := &model.MinionConnectHistory{
			MinionID:    id,
			Broker:      sessData.Broker,
			ExecuteStat: sessData.ExecuteStat,
			TunnelStat:  tunStatHis,
		}

		ctx, cancel := srv.perContext()
		defer cancel()
		hisColl := srv.db.MinionConnectHistory()
		_, _ = hisColl.InsertOne(ctx, his)
	}

	if ntf := srv.opts.Notifier; ntf == nil {
		srv.log().Debug("没有注册回调函数，无需触发下线通知回调")
	} else {
		srv.log().Debug("触发下线通知回调")

		ctx, cancel := srv.perContext()
		defer cancel()

		info := peer.Info()
		ntf.OnDisconnected(ctx, info, sessData.ConnectAt, sessData.DisconnectAt)
	}

	srv.log().Info("节点下线处理完毕", "session", sessData)
}
