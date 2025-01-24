package linkhub

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/vela-ssoc/vela-broker/app/temporary"
	"github.com/vela-ssoc/vela-broker/app/temporary/linkhub/concurrent"
	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mba/ciphertext"
	"github.com/vela-ssoc/vela-common-mba/definition"
	"github.com/xgfone/ship/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type minionHub struct {
	db        *gorm.DB
	qry       *query.Query
	broker    telecom.Linker
	sugar     logback.Logger
	outbox    chan *outboxMsg
	tokenSep  string
	random    *rand.Rand
	processes *concurrent.Map[temporary.Opcode, process]
	minions   concurrent.BucketMap[int64, *temporary.Conn]
	gfs       gridfs.FS
}

// New 新建 minion 消息处理器
func New(db *gorm.DB, qry *query.Query, brk telecom.Linker, sugar logback.Logger, gfs gridfs.FS) *minionHub {
	minions := concurrent.NewBucketMap[int64, *temporary.Conn](128, 32) // 128*32=4096
	processes := concurrent.NewMap[temporary.Opcode, process](16)
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	outbox := make(chan *outboxMsg, 128)

	// stm := stream.New(hcli, sugar)
	hub := &minionHub{
		db:        db,
		qry:       qry,
		broker:    brk,
		sugar:     sugar,
		outbox:    outbox,
		tokenSep:  ".",
		random:    random,
		processes: processes,
		minions:   minions,
		gfs:       gfs,
	}

	//proc := handle.New(db, ntm, sona, hub, bus, sugar)
	//_ = hub.AddProc(minion.OpHeartbeat, proc.Heartbeat)
	//_ = hub.AddProc(minion.OpTag, proc.Tag)
	//
	//_ = hub.AddProc(minion.OpRisk, proc.Risk)
	//_ = hub.AddProc(minion.OpSbom, proc.Sbom)
	//_ = hub.AddProc(minion.OpLogon, proc.Logon)
	//_ = hub.AddProc(minion.OpBulkES, proc.BulkES)
	//
	//_ = hub.AddProc(minion.OpEvent, proc.Event)
	//
	//_ = hub.AddProc(minion.OpAccountDiff, proc.AccountDiff)
	//_ = hub.AddProc(minion.OpAccountFull, proc.AccountFull)
	//_ = hub.AddProc(minion.OpProcessDiff, proc.ProcessDiff)
	//_ = hub.AddProc(minion.OpProcessFull, proc.ProcessFull)
	//_ = hub.AddProc(minion.OpGroupDiff, proc.ProcessDiff)
	//_ = hub.AddProc(minion.OpGroupFull, proc.GroupFull)
	//_ = hub.AddProc(minion.OpListenDiff, proc.ListenDiff)
	//_ = hub.AddProc(minion.OpListenFull, proc.ListenFull)
	//
	//_ = hub.AddProc(minion.OpSysInfo, proc.SysInfo)

	hub.worker(8)

	return hub
}

func (hub *minionHub) Authorize(ident temporary.Ident) (claim temporary.Claim, err error) {
	var inet string
	if ip4 := ident.Inet.To4(); ip4 != nil {
		inet = ip4.String()
	}
	var inet6 string
	if ip16 := ident.Inet6.To16(); ip16 != nil {
		inet6 = ip16.String()
	}
	mac := ident.MAC.String()

	// 通过inet查询节点信息
	var mn model.Minion
	if err = hub.db.Take(&mn, "inet = ?", inet).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			hub.sugar.Warnf("minion 节点新增错误：%v", err)
			err = ship.ErrBadRequest
			return
		}

		// 不存在则注册插入
		bident := hub.broker.Ident()
		issue := hub.broker.Issue()
		mn.Inet, mn.Inet6, mn.MAC = inet, inet6, mac
		mn.Goos, mn.Arch, mn.Edition = ident.Goos, ident.Arch, ident.Edition
		mn.BrokerID, mn.BrokerName, mn.Status = bident.ID, issue.Name, model.MSOffline
		// 创建数据、创建标签
		if err = hub.db.Create(&mn).Error; err != nil {
			return
		}
		mid := mn.ID
		tags := []*model.MinionTag{
			{Tag: inet, MinionID: mid, Kind: model.TkLifelong},
			{Tag: ident.Arch, MinionID: mid, Kind: model.TkLifelong},
			{Tag: ident.Goos, MinionID: mid, Kind: model.TkLifelong},
		}
		hub.db.Clauses(clause.OnConflict{DoNothing: true}).Create(tags)
	}
	if mn.Status == model.MSDelete { // 标记为删除则禁止登录
		err = ship.ErrForbidden
		return
	}
	if mn.Status == model.MSOnline { // 重复登录
		err = ship.ErrTooManyRequests
		return
	}

	token := hub.signToken(mn.ID)
	claim.ID, claim.Token, claim.Mask = mn.ID, token, hub.randMask()

	return
}

func (hub *minionHub) Timeout() time.Duration {
	return 15 * time.Minute
}

func (hub *minionHub) Connect(conn *temporary.Conn) {
	inet, minionID := conn.Inet(), conn.ID()
	ident := conn.Ident()
	var inet6 string
	if ipv6 := ident.Inet6.To16(); ipv6 != nil {
		inet6 = ipv6.String()
	}
	mac := ident.MAC.String()
	goos, arch, edition := ident.Goos, ident.Arch, ident.Edition

	brokerID, brokerName := hub.broker.Ident().ID, hub.broker.Issue().Name
	hub.sugar.Infof("minion 节点 %s (%d) 在 v2.0 接口上线", inet, minionID)

	hub.minions.Store(minionID, conn)
	columns := map[string]any{
		"broker_id": brokerID, "broker_name": brokerName, "status": model.MSOnline,
		"uptime": time.Now(), "inet6": inet6, "mac": mac, "goos": goos, "arch": arch, "edition": edition,
	}
	hub.db.Model(&model.Minion{}).Where("id = ? AND status = ?", minionID, model.MSOffline).
		Updates(columns)

	//ipv4 := inet.String()
	//subject := fmt.Sprintf("%s 在 %s 上线了", ipv4, brokerName)
	//evt := &model.Event{
	//	MinionID: minionID, Inet: ipv4, Subject: subject, Typeof: notice.EvtTypMinionConnect,
	//	Level: model.ELvlNote, OccurAt: time.Now(), CreatedAt: time.Now(),
	//}

	// 检查是否强制更新
	// var mbin model.MinionBin
	// err := hub.db.Find(&mbin, "goos = ? AND arch = ? AND semver = ?", goos, arch, edition).Error
	// if err == gorm.ErrRecordNotFound || mbin.Deprecated {

	// 强制升级
	_ = conn.Send(&temporary.Message{Opcode: temporary.OpUpgrade}) // 通知更新
	//}
}

func (hub *minionHub) Receive(conn *temporary.Conn, rec *temporary.Receive) {
	opcode := rec.Opcode()
	claim := conn.Claim()
	id, inet := claim.ID, conn.Inet() // 获取 minion ID
	hub.sugar.Debugf("收到 minion 节点 %s (%d) 的消息: %s", inet, id, opcode)

	proc, exist := hub.processes.Load(opcode)
	if !exist {
		hub.sugar.Warnf("minion 节点 %s (%d) [%s] 处理失败, 没有 process", inet, id, opcode)
		return
	}

	// 调用处理程序
	if err := proc.Invoke(conn, rec); err != nil {
		hub.sugar.Warnf("minion 节点 %s (%d) [%s] 处理失败: %v", inet, id, opcode, err)
	} else {
		hub.sugar.Infof("minion 节点 %s (%d) [%s] 处理成功", inet, id, opcode)
	}
}

func (hub *minionHub) Disconnect(conn *temporary.Conn) {
	claim := conn.Claim()
	inet := conn.Inet()
	minionID, brokerID := claim.ID, hub.broker.Ident().ID
	hub.sugar.Warnf("minion 节点 %s (%d) 下线了", inet, minionID)

	hub.minions.Delete(minionID)
	hub.db.Model(&model.Minion{}).
		Where("id = ? AND broker_id = ? AND status = ?", minionID, brokerID, model.MSOnline).
		Update("status", model.MSOffline)

	//ipv4 := inet.String()
	//subject := fmt.Sprintf("%s 在 %s 下线了", inet, brokerName)
	//evt := &model.Event{
	//	MinionID: minionID, Inet: ipv4, Subject: subject, Typeof: notice.EvtTypMinionDisconnect,
	//	Level: model.ELvlMajor, SendAlert: true, OccurAt: time.Now(), CreatedAt: time.Now(),
	//}
	//_ = hub.ntm.Event(evt)
}

func (hub *minionHub) TokenLookup(token string) *temporary.Conn {
	minionID := hub.parseID(token)
	mid, _ := strconv.ParseInt(minionID, 10, 64)

	if conn, exist := hub.minions.Load(mid); exist && conn.Authed(token) {
		return conn
	}

	return nil
}

func (hub *minionHub) StreamFunc(conn *temporary.Conn, ident temporary.StreamIdent) (temporary.StreamFunc, error) {
	// return hub.stream.Receive(conn, ident)
	return nil, io.ErrNoProgress
}

func (hub *minionHub) Unicast(minionID int64, opcode temporary.Opcode, data any) error {
	msg := &outboxMsg{MinionID: minionID, Opcode: opcode, Data: data}
	hub.outbox <- msg

	return nil
}

func (hub *minionHub) Multicast(minionIDs []int64, opcode temporary.Opcode, data any) error {
	hm := make(map[int64]struct{}, len(minionIDs))
	for _, minionID := range minionIDs {
		if _, exist := hm[minionID]; exist {
			continue
		}
		hm[minionID] = struct{}{}
		_ = hub.Unicast(minionID, opcode, data)
	}

	return nil
}

func (hub *minionHub) Broadcast(opcode temporary.Opcode, data any) error {
	msg := &outboxMsg{Broadcast: true, Opcode: opcode, Data: data}
	hub.outbox <- msg

	return nil
}

func (hub *minionHub) Reset() {
	bid := hub.broker.Ident().ID
	hub.db.Model(&model.Minion{}).
		Where("broker_id = ? AND status = ?", bid, model.MSOnline).
		Update("status", model.MSOffline)
}

// DelProc 删除对应 minion.Opcode 的处理程序
func (hub *minionHub) DelProc(opcode temporary.Opcode) {
	hub.processes.Delete(opcode)
}

// AddProc 为 minion.Opcode 添加对应的处理方法
func (hub *minionHub) AddProc(opcode temporary.Opcode, fn any) error {
	if fn == nil {
		return errors.New("fn方法不能为空")
	}

	connRef := reflect.TypeOf(new(temporary.Conn))
	errorRef := reflect.TypeOf((*error)(nil)).Elem()

	rvf := reflect.ValueOf(fn)
	if rvf.Kind() != reflect.Func {
		return fmt.Errorf("fn必须是%s类型", reflect.Func)
	}
	rtf := reflect.TypeOf(fn)
	nin, nou := rtf.NumIn(), rtf.NumOut()
	if nin != 1 && nin != 2 {
		return errors.New("方法参数必须是1-2个")
	}
	if nou != 1 {
		return errors.New("方法返回值必须是一个error")
	}
	if rtf.In(0) != connRef {
		return fmt.Errorf("第一个参数必须是%s类型", connRef)
	}
	if rtf.Out(0) != errorRef {
		return fmt.Errorf("方法返回值必须是%s类型", errorRef)
	}
	proc := process{fn: rvf}
	if nin == 2 {
		in, ptr := rtf.In(1), false
		if ptr = in.Kind() == reflect.Ptr; ptr {
			in = in.Elem()
		}

		proc.in, proc.arg, proc.ptr = in, true, ptr
	}

	hub.processes.Store(opcode, proc)

	return nil
}

// signToken 根据 minionID 生成 token
func (hub *minionHub) signToken(minionID int64) string {
	buf := make([]byte, 30)
	hub.random.Read(buf)
	mid := strconv.FormatInt(minionID, 10)
	return mid + hub.tokenSep + hex.EncodeToString(buf)
}

// parseID 从 token 中解析出 brokerID
func (hub *minionHub) parseID(token string) (minionID string) {
	if sn := strings.SplitN(token, hub.tokenSep, 2); len(sn) == 2 {
		minionID = sn[0]
	}
	return
}

// randMask 生成 mask (1-255)
func (hub *minionHub) randMask() byte {
	return byte(hub.random.Intn(0xff)) + 1
}

func (hub *minionHub) worker(n int) {
	for i := 0; i < n; i++ {
		go hub.work()
	}
}

func (hub *minionHub) work() {
	for msg := range hub.outbox {
		hub.send(msg)
	}
}

func (hub *minionHub) send(msg *outboxMsg) {
	opcode := msg.Opcode
	data := msg.Data
	if msg.IsBroadcast() {
		if err := hub.broadcast(opcode, data); err != nil {
			hub.sugar.Warnf("minion 广播消息 [%s] 发送失败: %v", opcode, err)
		} else {
			hub.sugar.Infof("minion 广播消息 [%s] 发送成功", opcode)
		}
	} else {
		minionID := msg.MinionID
		if err := hub.unicast(minionID, opcode, data); err != nil {
			hub.sugar.Warnf("向 minion: %d 发送 %s 消息失败: %v", minionID, opcode, err)
		} else {
			hub.sugar.Infof("向 minion: %d 发送 %s 消息成功", minionID, opcode)
		}
	}
}

func (hub *minionHub) unicast(minionID int64, opcode temporary.Opcode, data any) error {
	conn, exist := hub.minions.Load(minionID)
	if !exist {
		return errors.New("节点不在线")
	}
	return conn.Send(&temporary.Message{Opcode: opcode, Data: data})
}

func (hub *minionHub) broadcast(opcode temporary.Opcode, data any) error {
	sendFunc := func(_ int64, conn *temporary.Conn) { _ = conn.Send(&temporary.Message{Opcode: opcode, Data: data}) }
	hub.minions.Range(sendFunc)

	return nil
}

type upgradeRequest struct {
	Edition model.Semver `json:"edition" query:"edition" validate:"omitempty,semver"`
}

func (hub *minionHub) Upgrade(c *ship.Context) error {
	var req upgradeRequest
	if err := c.BindQuery(&req); err != nil {
		return err
	}

	conn := hub.TokenLookup(c.GetReqHeader(ship.HeaderAuthorization))
	if conn == nil {
		return c.NoContent(http.StatusUnauthorized)
	}
	ident := conn.Ident()
	goos, arch, version := ident.Goos, ident.Arch, ident.Edition

	ctx := c.Request().Context()
	db := hub.qry.MinionBin.WithContext(ctx).UnderlyingDB()

	stmt := db.Where("goos = ? AND arch = ?", goos, arch)
	if req.Edition == "" {
		current := model.Semver(version).Int64()
		stmt.Where("weight > ?", current).Order("weight DESC") // 查询最新版
	} else {
		if string(req.Edition) == version { // 无需更新
			return c.NoContent(http.StatusNoContent)
		}
		stmt.Where("semver = ?", req.Edition)
	}

	var edt model.MinionBin
	if err := stmt.First(&edt).Error; err != nil {
		// 没有找到比当前版本更高的无需更新
		return c.NoContent(http.StatusNoContent)
	}

	bid := hub.broker.Ident().ID
	// 查询 broker 信息
	brkTbl := hub.qry.Broker
	brk, err := brkTbl.WithContext(ctx).Where(brkTbl.ID.Eq(bid)).First()
	if err != nil {
		c.Warnf("更新版本查询 broker 信息错误：%s", err)
		return err
	}

	file, err := hub.gfs.OpenID(edt.FileID)
	if err != nil {
		c.Warnf("打开二进制文件错误：%s", err)
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer file.Close()

	addrs := make([]string, 0, 16)
	unique := make(map[string]struct{}, 16)
	for _, addr := range brk.LAN {
		if _, ok := unique[addr]; ok {
			continue
		}
		unique[addr] = struct{}{}
		addrs = append(addrs, addr)
	}
	for _, addr := range brk.VIP {
		if _, ok := unique[addr]; ok {
			continue
		}
		unique[addr] = struct{}{}
		addrs = append(addrs, addr)
	}
	hide := &definition.MHide{
		Servername: brk.Servername,
		Addrs:      addrs,
		Semver:     string(edt.Semver),
		Hash:       edt.Hash,
		Size:       edt.Size,
		Goos:       ident.Goos,
		Arch:       ident.Arch,
		DownloadAt: time.Now(),
		VIP:        brk.VIP,
		LAN:        brk.LAN,
		Edition:    string(edt.Semver),
	}

	enc, exx := ciphertext.EncryptPayload(hide)
	if err != nil {
		return exx
	}
	stm := gridfs.Merge(file, enc)

	// 此时的 Content-Length = 原始文件 + 隐藏文件
	c.Header().Set(ship.HeaderContentLength, stm.ContentLength())
	c.Header().Set(ship.HeaderContentDisposition, stm.Disposition())

	return c.Stream(http.StatusOK, stm.ContentType(), stm)
}
