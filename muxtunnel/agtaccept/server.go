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
	"runtime"
	"time"

	"github.com/vela-ssoc/ssoc-common/muxserver"
	"github.com/vela-ssoc/ssoc-common/store/model"
	"github.com/vela-ssoc/ssoc-common/store/repository"
	"github.com/vela-ssoc/ssoc-proto/muxconn"
	"github.com/vela-ssoc/ssoc-proto/muxproto"
	"github.com/vela-ssoc/ssoc-proto/muxtool"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Options struct {
	Huber      muxserver.Huber // 必须填写
	Handler    http.Handler
	Validator  func(any) error
	Logger     *slog.Logger
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
// 老版本的多路复用连接方式有诸多问题，将会逐渐淘汰掉。
func (srv *agentAccept) AcceptTCP(w http.ResponseWriter, r *http.Request) error {
	connectAt := time.Now()
	sessData := &tunnelSessionDataV1{
		Peer:          nil,
		Request:       nil,
		ConnectAt:     connectAt,
		LocalAddr:     "",
		RemoteAddr:    "",
		TunnelLibrary: model.TunnelLibrary{},
		ExecuteStat:   model.ExecuteStat{},
		TunnelStat:    model.TunnelStat{},
	}
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

	if runtime.NumCPU() != 100000000 { // FIXME 模拟错误
		pde := &muxtool.ProblemDetails{Status: http.StatusTooManyRequests, Detail: "限流测试"}
		srv.writeErrorV1(conn, r, pde) // 响应错误信息。

		return nil, errors.New(pde.Detail)
	}

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
		srv.log().Error(detail, "session", sessData)
		return nil, errors.New(detail)
	} else if node.Status == model.MinionStatusOnline {
		detail := "节点已经在线（数据库检查）"
		pde = &muxtool.ProblemDetails{Status: http.StatusConflict, Detail: detail}
		srv.log().Error(detail, "session", sessData)
		return nil, errors.New(detail)
	}

	return nil, nil
}

func (srv *agentAccept) findOrCreateV1(req *IdentV1) (*model.Minion, *muxtool.ProblemDetails) {
	//coll := srv.db.Minion()
	//coll.Find(ctx)
	//
	//machineID, inet := req.MachineID, req.Inet
	//if machineID != "" {
	//
	//}
	//
	//dat := &model.Minion{
	//	MachineID:   req.MachineID,
	//	Status:      0,
	//	Tags:        nil,
	//	TunnelStat:  nil,
	//	ExecuteStat: nil,
	//	CMDB:        nil,
	//	CreatedAt:   time.Time{},
	//	UpdatedAt:   time.Time{},
	//}

	return nil, nil
}

func (srv *agentAccept) updateOnline(data *tunnelSessionDataV1) error {
	//// 每次上线都要更新标签
	//goos, goarch, inet := req.Goos, req.Arch, req.Inet
	//tags := dat.Tags.ReplaceAllSystemTags(goos, goarch, inet)
	//
	//execStat := &model.ExecuteStat{
	//	Inet:       "",
	//	Goos:       "",
	//	Goarch:     "",
	//	Semver:     "",
	//	Version:    0,
	//	Unstable:   false,
	//	PID:        0,
	//	Args:       nil,
	//	Hostname:   "",
	//	Workdir:    "",
	//	Executable: "",
	//}
	//tunStat := &model.TunnelStat{
	//	ConnectedAt: time.Time{},
	//	KeepaliveAt: time.Time{},
	//	Library:     model.TunnelLibrary{},
	//	LocalAddr:   "",
	//	RemoteAddr:  "",
	//}
	//
	//filter := bson.M{"_id": id, "status": model.MinionStatusOffline}
	//update := bson.M{"$set": bson.M{
	//	"inet": inet,
	//
	//	"tags":     tags,
	//	"status":   model.MinionStatusOnline,
	//	"unstable": req.Unstable,
	//}}
	//type Minion struct {
	//	MachineID   string        `bson:"machine_id"             json:"machine_id"`
	//	Inet        string        `bson:"inet"                   json:"inet"` // 主要 IP，非唯一标识，主要用于用户识别的。
	//	Status      MinionStatus  `bson:"status"                 json:"status"`
	//	Tags        MinionTags    `bson:"tags"                   json:"tags"`   // 节点标签，配置下发。
	//	Unload      bool          `bson:"unload"                 json:"unload"` // 此模式开启，此节点不会加载任何配置。
	//	TunnelStat  *TunnelStat   `bson:"tunnel_stat,omitempty"  json:"tunnel_stat,omitzero"`
	//	ExecuteStat *ExecuteStat  `bson:"execute_stat,omitempty" json:"execute_stat,omitzero"`
	//	CMDB        *MinionCMDB   `bson:"cmdb,omitempty"         json:"cmdb,omitempty"`
	//	CreatedAt   time.Time     `bson:"created_at,omitempty"   json:"created_at"`
	//	UpdatedAt   time.Time     `bson:"updated_at,omitempty"   json:"updated_at"`
	//}

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
	_ = conn.SetWriteDeadline(time.Now().Add(d))
	res.Write(conn)
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
		filter := bson.D{{Key: "_id", Value: id}, {Key: "status", Value: true}}
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
			srv.log().Info("节点下线修改数据库完毕", "session", sessData)
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
