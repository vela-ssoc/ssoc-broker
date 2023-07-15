package agtapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/stegano"
	"github.com/vela-ssoc/vela-common-mba/definition"
	"github.com/xgfone/ship/v5"
	"gorm.io/gorm"
)

func Upgrade(bid int64, gfs gridfs.FS) route.Router {
	return &upgradeREST{
		bid:     bid,
		gfs:     gfs,
		maxsize: 100,
	}
}

type upgradeREST struct {
	bid     int64
	gfs     gridfs.FS
	mutex   sync.Mutex
	maxsize int
	count   int
}

func (rest *upgradeREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/upgrade/download").Data("agent 下载二进制文件升级").GET(rest.Download)
}

func (rest *upgradeREST) Download(c *ship.Context) error {
	if !rest.tryLock() {
		return c.NoContent(http.StatusTooManyRequests)
	}
	defer rest.unlock()

	var req param.UpgradeDownload
	if err := c.BindQuery(&req); err != nil {
		return err
	}

	r := c.Request()
	ctx := r.Context()
	inf := mlink.Ctx(r.Context()) // 获取节点的信息
	ident := inf.Ident()
	goos, arch := ident.Goos, ident.Arch
	except := req.Version
	if except != "" && except == ident.Semver { // 如果请求的版本
		c.WriteHeader(http.StatusNotModified)
		return nil
	}

	// 查询最新的版本信息
	tbl := query.MinionBin
	dao := tbl.WithContext(ctx).
		Where(tbl.Goos.Eq(goos),
			tbl.Arch.Eq(arch),
			tbl.Deprecated.Is(false)).
		Order(tbl.Weight.Desc(), tbl.UpdatedAt.Desc())
	if except != "" {
		dao.Where(tbl.Semver.Eq(except))
	} else {
		weight := model.Semver(ident.Semver).Int64()
		dao.Where(tbl.Weight.Gt(weight))
	}

	bin, err := dao.First()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.WriteHeader(http.StatusNotModified)
			return nil
		}
		c.Warnf("查询更新版本出错：%s", err)
		return err
	}

	// 查询 broker 信息
	brkTbl := query.Broker
	brk, err := brkTbl.WithContext(ctx).Where(brkTbl.ID.Eq(rest.bid)).First()
	if err != nil {
		c.Warnf("更新版本查询 broker 信息错误：%s", err)
		return err
	}

	file, err := rest.gfs.OpenID(bin.FileID)
	if err != nil {
		c.Warnf("打开二进制文件错误：%s", err)
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer file.Close()

	hide := &definition.MinionHide{
		Servername: brk.Servername,
		LAN:        brk.LAN,
		VIP:        brk.VIP,
		Edition:    string(bin.Semver),
		Hash:       file.MD5(),
		Size:       file.Size(),
		Tags:       req.Tags,
		DownloadAt: time.Now(),
	}
	stm, err := stegano.AppendStream(file, hide)
	if err != nil {
		return err
	}

	// 此时的 Content-Length = 原始文件 + 隐藏文件
	c.Header().Set(ship.HeaderContentLength, stm.ContentLength())
	c.Header().Set(ship.HeaderContentDisposition, stm.Disposition())

	return c.Stream(http.StatusOK, stm.ContentType(), stm)
}

func (rest *upgradeREST) tryLock() bool {
	rest.mutex.Lock()
	defer rest.mutex.Unlock()

	ok := rest.maxsize > rest.count
	if ok {
		rest.count++
	}

	return ok
}

func (rest *upgradeREST) unlock() {
	rest.mutex.Lock()
	defer rest.mutex.Unlock()
	rest.count--
	if rest.count < 0 {
		rest.count = 0
	}
}
