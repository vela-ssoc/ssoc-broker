package mlink

import (
	"context"
	"database/sql"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mb/problem"
	"github.com/vela-ssoc/vela-common-mba/netutil"
	"github.com/vela-ssoc/vela-common-mba/smux"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrMinionBadInet  = errors.New("minion IP 不合法")
	ErrMinionInactive = errors.New("节点未激活")
	ErrMinionRemove   = errors.New("节点已删除")
	ErrMinionOnline   = errors.New("节点已经在线")
	ErrMinionOffline  = errors.New("节点未在线")
)

type Linker interface {
	ResetDB() error
	gateway.Joiner
	Huber
	Link() telecom.Linker
}

type Huber interface {
	Forward(http.ResponseWriter, *http.Request)

	Stream(ctx context.Context, id int64, path string, header http.Header) (*websocket.Conn, *http.Response, error)

	Oneway(id int64, path string, body any) error

	Unicast(ctx context.Context, id int64, path string, body, resp any) error

	Multicast(ids []int64, path string, body any) <-chan *Future

	Broadcast(path string, body any) <-chan *Future

	// Knockout 根据 minionID 断开节点连接
	Knockout(mid int64)
}

func LinkHub(link telecom.Linker, handler http.Handler, phase NodePhaser, pool gopool.Executor) Linker {
	seed := time.Now().UnixNano()
	random := rand.New(rand.NewSource(seed))

	ph := newAsyncPhase(phase, pool)

	hub := &minionHub{
		link:    link,
		handler: handler,
		bid:     link.Ident().ID,
		name:    link.Name(),
		section: container(),
		phase:   ph,
		random:  random,
		pool:    pool,
	}

	trip := &http.Transport{DialContext: hub.dialContext}
	hub.client = netutil.NewClient(trip)
	hub.stream = netutil.NewStream(hub.dialContext)
	hub.proxy = netutil.NewForward(trip, hub.forwardError)

	return hub
}

type minionHub struct {
	link    telecom.Linker
	handler http.Handler
	slog    logback.Logger
	client  netutil.HTTPClient
	proxy   netutil.Forwarder
	stream  netutil.Streamer
	phase   NodePhaser
	pool    gopool.Executor
	section subsection
	bid     int64  // 当前 broker ID
	name    string // 当前 broker 名字
	random  *rand.Rand
}

func (hub *minionHub) Link() telecom.Linker {
	return hub.link
}

func (hub *minionHub) Auth(ctx context.Context, ident gateway.Ident) (gateway.Issue, http.Header, bool, error) {
	var issue gateway.Issue
	ip := ident.Inet.To4()
	if ip == nil || ip.IsLoopback() || ip.IsUnspecified() {
		return issue, nil, false, ErrMinionBadInet
	}

	// 根据 inet 查询节点信息
	now := time.Now()
	inet := ip.String()
	monTbl := query.Minion
	mon, err := monTbl.WithContext(ctx).Where(monTbl.Inet.Eq(inet)).First()
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return issue, nil, false, err
		}

		join := &model.Minion{
			Inet: inet,
			// Name:       inet,
			MAC:    ident.MAC,
			Goos:   ident.Goos,
			Arch:   ident.Arch,
			Status: model.MSOffline,
			// Semver:     ident.Semver,
			// CPU:        ident.CPU,
			// PID:        ident.PID,
			// Username:   ident.Username,
			// Hostname:   ident.Hostname,
			// Workdir:    ident.Workdir,
			// Executable: ident.Executable,
			// JoinedAt:   now,
			Unload:     ident.Unload,
			BrokerID:   hub.link.Ident().ID,
			BrokerName: hub.link.Issue().Name,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err = query.Q.Transaction(func(tx *query.Query) error {
			if exx := tx.WithContext(ctx).Minion.Create(join); exx != nil {
				return exx
			}
			tags := model.MinionTags{
				{Tag: ident.Goos, MinionID: join.ID, Kind: model.TkLifelong},
				{Tag: ident.Arch, MinionID: join.ID, Kind: model.TkLifelong},
				{Tag: inet, MinionID: join.ID, Kind: model.TkLifelong},
			}
			return tx.WithContext(ctx).MinionTag.
				Clauses(clause.OnConflict{DoNothing: true}).
				Create(tags...)
		}); err != nil {
			return issue, nil, false, err
		}

		hub.phase.Created(join.ID, inet, now)
	}

	status := mon.Status
	if status == model.MSInactive {
		return issue, nil, false, ErrMinionInactive
	}
	if status == model.MSDelete {
		return issue, nil, true, ErrMinionRemove
	}
	if status == model.MSOnline {
		return issue, nil, false, ErrMinionOnline
	}

	issue.ID = mon.ID
	if ident.Encrypt {
		// 随机生成一个 32-64 位长度的加密密钥
		psz := hub.random.Intn(33) + 32
		passwd := make([]byte, psz)
		hub.random.Read(passwd)
		issue.Passwd = passwd
	}

	return issue, nil, false, nil
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

	id := issue.ID
	inet := ident.Inet.String()
	before := time.Now()
	sid := strconv.FormatInt(id, 10) // 方便 dialContext
	conn := &connect{
		id:    id,
		ident: ident,
		issue: issue,
		mux:   mux,
	}

	if !hub.section.put(sid, conn) {
		hub.phase.Repeated(id, ident, before)
		return ErrMinionOnline
	}
	defer hub.section.del(sid)

	now := sql.NullTime{Valid: true, Time: time.Now()}
	mon := &model.Minion{
		ID:      id,
		Inet:    inet,
		Status:  model.MSOnline,
		MAC:     ident.MAC,
		Goos:    ident.Goos,
		Arch:    ident.Arch,
		Edition: ident.Semver,
		// CPU:        ident.CPU,
		// PID:        ident.PID,
		// Username:   ident.Username,
		// Hostname:   ident.Hostname,
		// Workdir:    ident.Workdir,
		// Executable: ident.Executable,
		// PingedAt:   now,
		// JoinedAt:   now,
		Uptime:     now,
		BrokerID:   hub.link.Ident().ID,
		BrokerName: hub.link.Issue().Name,
		UpdatedAt:  before,
	}

	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	monTbl := query.Minion
	info, err := monTbl.WithContext(ctx).
		Where(monTbl.ID.Eq(id), monTbl.Status.Eq(uint8(model.MSOffline))).
		UpdateColumns(mon)
	cancel()
	if err != nil {
		hub.slog.Warnf("节点 %s(%d) 上线状态更新错误：%s", inet, id, err)
		return err
	}
	if info.RowsAffected == 0 {
		hub.slog.Warnf("节点 %s(%d) 上线状态未发生更新", inet, id)
		return ErrMinionOnline
	}

	defer func() {
		online := uint8(model.MSOnline)
		offline := uint8(model.MSOffline)
		dctx, dcancel := context.WithTimeout(parent, 10*time.Second)
		_, exx := monTbl.WithContext(dctx).
			Where(monTbl.ID.Eq(id)).
			Where(monTbl.BrokerID.Eq(hub.bid)).
			Where(monTbl.Status.Eq(online)).
			UpdateColumnSimple(monTbl.Status.Value(offline))
		dcancel()
		if exx != nil {
			hub.slog.Warnf("节点 %s(%d) 修改下线状态错误: %s", inet, id, exx)
		}
	}()

	srv := &http.Server{
		Handler: hub.handler,
		BaseContext: func(net.Listener) context.Context {
			return context.WithValue(context.Background(), minionCtxKey, conn)
		},
	}

	hub.phase.Connected(hub, ident, issue, before)
	_ = srv.Serve(mux)
	after := time.Now()
	du := after.Sub(before)
	hub.phase.Disconnected(hub, ident, issue, after, du)

	return nil
}

func (hub *minionHub) Name() string {
	return hub.name
}

func (hub *minionHub) ResetDB() error {
	online := uint8(model.MSOnline)
	offline := uint8(model.MSOffline)
	tbl := query.Minion
	_, err := tbl.WithContext(context.Background()).
		Where(tbl.BrokerID.Eq(hub.bid)).
		Where(tbl.Status.Eq(online)).
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

func (hub *minionHub) Oneway(id int64, path string, body any) error {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	tsk := &onewayTask{
		wg:   wg,
		hub:  hub,
		mid:  id,
		path: path,
		req:  body,
	}
	hub.pool.Submit(tsk)
	return tsk.Wait()
}

func (hub *minionHub) Unicast(ctx context.Context, id int64, path string, body, resp any) error {
	wg := new(sync.WaitGroup)
	wg.Add(1)

	tsk := &unicastTask{
		wg:   wg,
		hub:  hub,
		mid:  id,
		path: path,
		req:  body,
		resp: resp,
	}
	hub.pool.Submit(tsk)

	return tsk.Wait()
}

func (hub *minionHub) Multicast(ids []int64, path string, body any) <-chan *Future {
	size := len(ids)
	ret := make(chan *Future, size)
	go hub.multicast(ids, path, body, ret)

	return ret
}

func (hub *minionHub) multicast(ids []int64, path string, body any, ret chan *Future) {
	wg := new(sync.WaitGroup)
	for _, id := range ids {
		tsk := &futureTask{
			wg:   wg,
			hub:  hub,
			mid:  id,
			path: path,
			req:  body,
			ret:  ret,
		}
		wg.Add(1)
		hub.pool.Submit(tsk)
	}

	wg.Wait()
	close(ret)
}

func (hub *minionHub) Broadcast(path string, body any) <-chan *Future {
	ret := make(chan *Future, 100)
	go hub.broadcast(path, body, ret)

	return ret
}

func (hub *minionHub) broadcast(path string, body any, ret chan *Future) {
	wg := new(sync.WaitGroup)
	iter := hub.section.iterator()
	for iter.has() {
		ids := iter.next()
		if len(ids) == 0 {
			continue
		}
		for _, id := range ids {
			tsk := &futureTask{
				wg:   wg,
				hub:  hub,
				mid:  id,
				path: path,
				req:  body,
				ret:  ret,
			}
			wg.Add(1)
			hub.pool.Submit(tsk)
		}
	}

	wg.Wait()
	close(ret)
}

func (hub *minionHub) Knockout(mid int64) {
	if mid == 0 {
		return
	}

	id := strconv.FormatInt(mid, 10)
	if conn := hub.section.del(id); conn != nil {
		_ = conn.mux.Close()
	}
}

func (hub *minionHub) silentJSON(ctx context.Context, id int64, path string, req any) error {
	addr := hub.httpURL(id, path)
	return hub.client.SilentJSON(ctx, http.MethodPost, addr, req, nil)
}

func (hub *minionHub) json(ctx context.Context, id int64, path string, req, resp any) error {
	addr := hub.httpURL(id, path)
	return hub.client.JSON(ctx, http.MethodPost, addr, req, resp, nil)
}

func (hub *minionHub) httpURL(id int64, path string) string {
	u := &url.URL{Scheme: "http", Host: strconv.FormatInt(id, 10), Path: path}
	return u.String()
}

func (hub *minionHub) wsURL(id int64, path string) string {
	u := &url.URL{Scheme: "ws", Host: strconv.FormatInt(id, 10), Path: path}
	return u.String()
}

func (hub *minionHub) dialContext(_ context.Context, _, addr string) (net.Conn, error) {
	id, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, net.InvalidAddrError(addr)
	}

	if conn, ok := hub.section.get(id); ok {
		if stream, exx := conn.mux.OpenStream(); exx != nil {
			return nil, exx
		} else {
			return stream, nil
		}
	}

	return nil, ErrMinionOffline
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
