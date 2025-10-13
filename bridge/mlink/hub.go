package mlink

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/ssoc-broker/bridge/gateway"
	"github.com/vela-ssoc/ssoc-broker/bridge/telecom"
	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/dal/query"
	"github.com/vela-ssoc/ssoc-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/vela-ssoc/vela-common-mba/smux"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrMinionBadInet   = errors.New("节点 IP 不合法")
	ErrMinionMachineID = errors.New("无效的机器码")
	ErrMinionInactive  = errors.New("节点未激活")
	ErrMinionRemove    = errors.New("节点已删除")
	ErrMinionOnline    = errors.New("节点已经在线")
	ErrMinionOffline   = errors.New("节点未在线")
)

type Linker interface {
	ResetDB() error
	gateway.Joiner
	Huber
	Link() telecom.Linker
}

type Huber interface {
	ConnectIDs() []int64

	Forward(http.ResponseWriter, *http.Request)

	Stream(ctx context.Context, id int64, path string, header http.Header) (*websocket.Conn, *http.Response, error)

	Oneway(ctx context.Context, id int64, path string, body any) error

	Unicast(ctx context.Context, id int64, path string, body, resp any) error

	// Knockout 根据 minionID 断开节点连接
	Knockout(mid int64)
}

func LinkHub(qry *query.Query, link telecom.Linker, handler http.Handler, phase NodePhaser, log *slog.Logger) Linker {
	seed := time.Now().UnixNano()
	random := rand.New(rand.NewSource(seed))

	hub := &minionHub{
		qry:     qry,
		link:    link,
		handler: handler,
		bid:     link.Ident().ID,
		name:    link.Name(),
		log:     log,
		section: newSegmentMap(128, 64), // 预分配 8192 个连接空间，已经足够使用了。
		phase:   phase,
		random:  random,
	}

	trip := &http.Transport{DialContext: hub.dialContext}
	hub.client = netutil.NewClient(trip)
	hub.stream = netutil.NewStream(hub.dialContext)
	hub.proxy = netutil.NewForward(trip, hub.forwardError)

	return hub
}

type minionHub struct {
	qry     *query.Query
	link    telecom.Linker
	handler http.Handler
	log     *slog.Logger
	client  netutil.HTTPClient
	proxy   netutil.Forwarder
	stream  netutil.Streamer
	phase   NodePhaser
	section container
	bid     int64  // 当前 broker ID
	name    string // 当前 broker 名字
	random  *rand.Rand
}

func (hub *minionHub) Link() telecom.Linker {
	return hub.link
}

func (hub *minionHub) Auth(ctx context.Context, ident gateway.Ident) (gateway.Issue, http.Header, int, error) {
	var issue gateway.Issue
	machineID := ident.MachineID
	if machineID == "" {
		return issue, nil, http.StatusBadRequest, ErrMinionMachineID
	}

	ip := ident.Inet.To4()
	if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
		return issue, nil, http.StatusBadRequest, ErrMinionBadInet
	}

	// 根据 inet 查询节点信息
	inet := ip.String()
	monTbl := hub.qry.Minion
	monDao := monTbl.WithContext(ctx)

	// 先通过机器码查询，如果查询不到再通过 inet 查询（兼容老的方式）。
	// 如果通过 inet 查询出来并且 machine_id 为空，则进行合并
	mon, err := monDao.Where(monTbl.MachineID.Eq(machineID)).First()
	if errors.Is(err, gorm.ErrRecordNotFound) { // 通过机器
		mon, err = monDao.Where(monTbl.Inet.Eq(inet), monTbl.MachineID.Eq("")).First()
		if err == nil { // 通过 Inet 找到了，兼容老的 agent 绑定机器码
			if _, err = monDao.Where(monTbl.ID.Eq(mon.ID)).
				UpdateColumnSimple(monTbl.MachineID.Value(machineID)); err != nil {
				return issue, nil, http.StatusInternalServerError, err
			}
			mon.MachineID = machineID
		}
	}
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return issue, nil, http.StatusBadRequest, err
		}

		newData, err := hub.createNew(ctx, ident)
		if err != nil {
			return issue, nil, http.StatusInternalServerError, err
		}
		mon = newData
	}

	status := mon.Status
	if status == model.MSDelete {
		return issue, nil, http.StatusForbidden, ErrMinionRemove
	}
	if status == model.MSOnline {
		return issue, nil, http.StatusConflict, ErrMinionOnline
	}

	issue.ID = mon.ID
	// 随机生成一个 32-64 位长度的加密密钥
	psz := hub.random.Intn(33) + 32
	passwd := make([]byte, psz)
	hub.random.Read(passwd)
	issue.Passwd = passwd

	return issue, nil, http.StatusAccepted, nil
}

func (hub *minionHub) Join(parent context.Context, tran net.Conn, ident gateway.Ident, issue gateway.Issue) error {
	cfg := smux.DefaultConfig()
	cfg.Passwd = issue.Passwd
	if inter := ident.Interval; inter > 0 {
		cfg.ReadTimeout = 3 * inter // 3 倍心跳周期还未收到消息，强制断开连接
	}
	mux := smux.Server(tran, cfg)
	//goland:noinspection GoUnhandledErrorResult
	defer mux.Close()

	id, machineID := issue.ID, ident.MachineID
	inet := ident.Inet.String()
	now := time.Now()
	sid := strconv.FormatInt(id, 10) // 方便 dialContext
	conn := &connect{
		id:    id,
		ident: ident,
		issue: issue,
		mux:   mux,
	}

	if !hub.section.Put(sid, conn) {
		hub.phase.Repeated(id, ident, now)
		return ErrMinionOnline
	}
	defer hub.section.Del(sid)

	nullableAt := sql.NullTime{Valid: true, Time: now}
	brokerID, brokerName := hub.link.Ident().ID, hub.link.Issue().Name
	online, offline := uint8(model.MSOnline), uint8(model.MSOffline)

	minionTbl := hub.qry.Minion
	{
		ctx, cancel := context.WithTimeout(parent, 10*time.Second)
		info, err := minionTbl.WithContext(ctx).
			Where(minionTbl.ID.Eq(id), minionTbl.Status.Eq(offline)).
			UpdateSimple(
				minionTbl.Status.Value(online),
				minionTbl.MAC.Value(ident.MAC),
				minionTbl.Goos.Value(ident.Goos),
				minionTbl.Arch.Value(ident.Arch),
				minionTbl.Edition.Value(ident.Semver),
				minionTbl.Unstable.Value(ident.Unstable),
				minionTbl.Customized.Value(ident.Customized),
				minionTbl.Uptime.Value(nullableAt),
				minionTbl.BrokerID.Value(brokerID),
				minionTbl.BrokerName.Value(brokerName),
			)
		cancel()
		if err != nil || info.RowsAffected == 0 {
			hub.log.Warn(fmt.Sprintf("节点 %s(%d) 修改上线状态失败：%v", inet, id, err))
			return err
		}
	}
	defer func() {
		dctx, dcancel := context.WithTimeout(context.Background(), time.Minute)
		ret, exx := minionTbl.WithContext(dctx).
			Where(minionTbl.MachineID.Eq(machineID)).
			Where(minionTbl.BrokerID.Eq(hub.bid)).
			Where(minionTbl.Status.Eq(online)).
			UpdateSimple(minionTbl.Status.Value(offline))
		dcancel()
		if exx != nil || ret.RowsAffected == 0 {
			hub.log.Warn(fmt.Sprintf("节点 %s(%d) 修改下线状态错误: %v", inet, id, exx))
		} else {
			hub.log.Info(fmt.Sprintf("节点 %s(%d) 修改下线状态成功", inet, id))
		}
	}()

	{
		ctx, cancel := context.WithTimeout(parent, 20*time.Second)
		// 每次上线都要重新初始化内置标签，长时间运行后，服务器可能重装系统。
		_ = hub.qry.Transaction(func(tx *query.Query) error {
			const kind = model.TkLifelong
			tags := model.MinionTags{
				&model.MinionTag{MinionID: id, Tag: inet, Kind: kind},
			}
			if goos := ident.Goos; goos != "" {
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
			_ = dao.Clauses(clause.OnConflict{DoNothing: true}).Create(tags...)

			return nil
		})

		cancel()
	}
	srv := &http.Server{
		Handler: hub.handler,
		BaseContext: func(net.Listener) context.Context {
			return context.WithValue(context.Background(), minionCtxKey, conn)
		},
	}

	hub.phase.Connected(hub, ident, issue, now)
	_ = srv.Serve(mux)
	after := time.Now()
	du := after.Sub(now)
	hub.phase.Disconnected(hub, ident, issue, after, du)

	return nil
}

func (hub *minionHub) Name() string {
	return hub.name
}

func (hub *minionHub) ResetDB() error {
	online := uint8(model.MSOnline)
	offline := uint8(model.MSOffline)
	tbl := hub.qry.Minion

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	_, err := tbl.WithContext(ctx).
		Where(tbl.BrokerID.Eq(hub.bid), tbl.Status.Eq(online)).
		UpdateColumnSimple(tbl.Status.Value(offline))
	return err
}

func (hub *minionHub) Forward(w http.ResponseWriter, r *http.Request) {
	hub.proxy.Forward(w, r)
}

func (hub *minionHub) Stream(ctx context.Context, id int64, path string, header http.Header) (*websocket.Conn, *http.Response, error) {
	addr := hub.wsURL(id, path)
	return hub.stream.Stream(ctx, addr, header)
}

func (hub *minionHub) Oneway(ctx context.Context, id int64, path string, body any) error {
	res, err := hub.sendJSON(ctx, id, path, body)
	if err == nil {
		_ = res.Body.Close()
	}
	return err
}

func (hub *minionHub) Unicast(ctx context.Context, id int64, path string, body, resp any) error {
	res, err := hub.sendJSON(ctx, id, path, body)
	if err != nil {
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer res.Body.Close()

	return json.NewDecoder(res.Body).Decode(resp)
}

func (hub *minionHub) ConnectIDs() []int64 {
	return hub.section.IDs()
}

func (hub *minionHub) Knockout(mid int64) {
	if mid == 0 {
		return
	}

	id := strconv.FormatInt(mid, 10)
	if conn := hub.section.Del(id); conn != nil {
		_ = conn.mux.Close()
	}
}

func (hub *minionHub) sendJSON(ctx context.Context, id int64, path string, req any) (*http.Response, error) {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
	}

	addr := hub.httpURL(id, path)

	return hub.client.DoJSON(ctx, http.MethodPost, addr, req, nil)
}

func (hub *minionHub) httpURL(id int64, path string) string {
	return hub.newURL(id, "http", path)
}

func (hub *minionHub) wsURL(id int64, path string) string {
	return hub.newURL(id, "ws", path)
}

func (*minionHub) newURL(id int64, scheme, path string) string {
	sid := strconv.FormatInt(id, 10)
	sn := strings.SplitN(path, "?", 2)
	u := &url.URL{Scheme: scheme, Host: sid, Path: sn[0]}
	if len(sn) == 2 {
		u.RawQuery = sn[1]
	}
	return u.String()
}

func (hub *minionHub) dialContext(_ context.Context, _, addr string) (net.Conn, error) {
	id, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, net.InvalidAddrError(addr)
	}

	conn := hub.section.Get(id)
	if conn == nil {
		return nil, ErrMinionOffline
	}

	if stream, exx := conn.mux.OpenStream(); exx != nil {
		return nil, exx
	} else {
		return stream, nil
	}
}

func (hub *minionHub) forwardError(w http.ResponseWriter, r *http.Request, err error) {
	pd := &problem.Detail{
		Type:     hub.name,
		Title:    "网关错误",
		Status:   http.StatusBadRequest,
		Detail:   err.Error(),
		Instance: r.RequestURI,
	}

	_ = pd.JSON(w)
}

func (hub *minionHub) createNew(ctx context.Context, ident gateway.Ident) (*model.Minion, error) {
	data := &model.Minion{
		MachineID:  ident.MachineID,
		Inet:       ident.Inet.String(),
		MAC:        ident.MAC,
		Goos:       ident.Goos,
		Arch:       ident.Arch,
		Edition:    ident.Semver,
		Status:     model.MSOffline,
		Uptime:     sql.NullTime{Time: time.Now(), Valid: true},
		BrokerID:   hub.bid,
		BrokerName: hub.name,
		Unload:     ident.Unload,
		Unstable:   ident.Unstable,
		Customized: ident.Customized,
	}

	tbl := hub.qry.Minion
	dao := tbl.WithContext(ctx)
	if err := dao.Create(data); err != nil {
		return nil, err
	}

	return data, nil
}
