package agtapi

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mba/ciphertext"
	"github.com/vela-ssoc/vela-common-mba/definition"
	"github.com/xgfone/ship/v5"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func Upgrade(bid int64, gfs gridfs.FS) route.Router {
	return &upgradeREST{
		bid:     bid,
		gfs:     gfs,
		maxsize: 200,
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
	except := req.Version // 期望升级到的版本
	if req.Unstable == ident.Unstable &&
		req.Customized == req.Customized &&
		except == ident.Semver {
		c.WriteHeader(http.StatusNotModified)
		return nil
	}

	// 查询 broker 信息
	brkTbl := query.Broker
	brk, err := brkTbl.WithContext(ctx).Where(brkTbl.ID.Eq(rest.bid)).First()
	if err != nil {
		c.Warnf("更新版本查询 broker 信息错误：%s", err)
		return err
	}

	tbl := query.MinionBin
	cond := []gen.Condition{
		tbl.Goos.Eq(ident.Goos),
		tbl.Arch.Eq(ident.Arch),
		tbl.Unstable.Is(req.Unstable),
	}
	if except != "" {
		cond = append(cond, tbl.Semver.Eq(except))
	}
	customized := req.Customized
	if customized == "" {
		customized = ident.Customized
	}
	cond = append(cond, tbl.Customized.Eq(customized))

	bin, err := tbl.WithContext(ctx).Where(cond...).Order(tbl.Weight.Desc()).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.WriteHeader(http.StatusNotModified)
			return nil
		}
		c.Warnf("查询更新版本出错：%s", err)
		return err
	}

	file, err := rest.gfs.OpenID(bin.FileID)
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
		Semver:     string(bin.Semver),
		Hash:       bin.Hash,
		Size:       bin.Size,
		Tags:       req.Tags,
		Goos:       bin.Goos,
		Arch:       bin.Arch,
		Unstable:   bin.Unstable,
		Customized: bin.Customized,
		DownloadAt: time.Now(),
		VIP:        brk.VIP,
		LAN:        brk.LAN,
		Edition:    string(bin.Semver),
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

func (rest *upgradeREST) suitableMinion(ctx context.Context, goos, arch, version string) (*model.MinionBin, error) {
	tbl := query.MinionBin
	dao := tbl.WithContext(ctx).
		Where(tbl.Deprecated.Is(false), tbl.Goos.Eq(goos), tbl.Arch.Eq(arch)).
		Order(tbl.Weight.Desc(), tbl.UpdatedAt.Desc())
	if version != "" {
		return dao.Where(tbl.Semver.Eq(version)).First()
	}

	// 版本号包含 - + 的权重会下降，例如：
	// 0.0.1-debug < 0.0.1
	// 0.0.1+20230720 < 0.0.1
	return dao.Where(tbl.Semver.NotLike("%-%"), tbl.Semver.NotLike("%+%")).
		First()
}
