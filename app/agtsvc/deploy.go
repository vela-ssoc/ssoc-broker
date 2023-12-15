package agtsvc

import (
	"context"
	"io"
	"time"

	"github.com/vela-ssoc/vela-broker/app/internal/modview"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/storage/v2"
	"github.com/vela-ssoc/vela-common-mba/ciphertext"
	"github.com/vela-ssoc/vela-common-mba/definition"
	"gorm.io/gen"
)

type DeployService interface {
	Script(ctx context.Context, goos string, data *modview.Deploy) (io.Reader, error)
	OpenMinion(ctx context.Context, req *param.DeployMinionDownload) (gridfs.File, error)
}

func Deploy(store storage.Storer, gfs gridfs.FS) DeployService {
	return &deployService{
		store: store,
		gfs:   gfs,
	}
}

type deployService struct {
	store storage.Storer
	gfs   gridfs.FS
}

func (biz *deployService) OpenMinion(ctx context.Context, req *param.DeployMinionDownload) (gridfs.File, error) {
	// 查询 broker 节点信息
	brkTbl := query.Broker
	brk, err := brkTbl.WithContext(ctx).Where(brkTbl.ID.Eq(req.BrokerID)).First()
	if err != nil {
		return nil, err
	}

	// 根据输入条件匹配合适版本
	tbl := query.MinionBin
	cond := []gen.Condition{
		tbl.Unstable.Is(req.Unstable),
		tbl.Customized.Eq(req.Customized),
		tbl.Deprecated.Is(false),
	}
	if req.ID != 0 {
		cond = append(cond, tbl.ID.Eq(req.ID))
	} else {
		cond = append(cond, tbl.Goos.Eq(req.Goos))
		cond = append(cond, tbl.Arch.Eq(req.Arch))
	}
	if ver := req.Version; ver != "" {
		cond = append(cond, tbl.Semver.Eq(string(ver)))
	}

	bin, err := tbl.WithContext(ctx).
		Where(cond...).
		Order(tbl.Weight.Desc()).
		First()
	if err != nil {
		return nil, err
	}

	inf, err := biz.gfs.OpenID(bin.FileID)
	if err != nil {
		return nil, err
	}

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
		Unload:     req.Unload,
		Unstable:   req.Unstable,
		Customized: req.Customized,
		DownloadAt: time.Now(),
		VIP:        brk.VIP,
		LAN:        brk.LAN,
		Edition:    string(bin.Semver),
	}

	enc, exx := ciphertext.EncryptPayload(hide)
	if exx != nil {
		_ = inf.Close()
		return nil, exx
	}

	file := gridfs.Merge(inf, enc)

	return file, nil
}

func (biz *deployService) Script(ctx context.Context, goos string, data *modview.Deploy) (io.Reader, error) {
	buf := biz.store.DeployScript(ctx, goos, data)
	return buf, nil
}

func (biz *deployService) suitableMinion(ctx context.Context, id int64, goos, arch, version string) (*model.MinionBin, error) {
	tbl := query.MinionBin
	dao := tbl.WithContext(ctx).
		Where(tbl.Deprecated.Is(false)).
		Order(tbl.Weight.Desc(), tbl.UpdatedAt.Desc())
	if id != 0 {
		return dao.Where(tbl.ID.Eq(id)).First()
	}

	if version != "" {
		return dao.Where(tbl.Goos.Eq(goos), tbl.Arch.Eq(arch), tbl.Semver.Eq(version)).First()
	}

	// 版本号包含 - + 的权重会下降，例如：
	// 0.0.1-debug < 0.0.1
	// 0.0.1+20230720 < 0.0.1
	stmt := dao.Where(tbl.Goos.Eq(goos), tbl.Arch.Eq(arch))
	bin, err := stmt.WithContext(ctx).
		Where(tbl.Semver.NotLike("%-%"), tbl.Semver.NotLike("%+%")).
		First()
	if err == nil {
		return bin, nil
	}

	return stmt.First()
}
