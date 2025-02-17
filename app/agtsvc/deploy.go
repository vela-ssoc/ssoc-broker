package agtsvc

import (
	"context"
	"errors"
	"io"
	"time"

	"gorm.io/gen/field"

	"github.com/vela-ssoc/vela-broker/app/internal/modview"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-common-mb-itai/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb-itai/dal/model"
	"github.com/vela-ssoc/vela-common-mb-itai/dal/query"
	"github.com/vela-ssoc/vela-common-mb-itai/storage/v2"
	"github.com/vela-ssoc/vela-common-mba/ciphertext"
	"github.com/vela-ssoc/vela-common-mba/definition"
	"gorm.io/gen"
)

type DeployService interface {
	Script(ctx context.Context, goos string, data *modview.Deploy) (io.Reader, error)
	OpenMinion(ctx context.Context, req *param.DeployMinionDownload) (gridfs.File, error)
}

func Deploy(store storage.Storer, gfs gridfs.FS, bid int64) DeployService {
	return &deployService{
		store: store,
		gfs:   gfs,
		bid:   bid,
	}
}

type deployService struct {
	store storage.Storer
	gfs   gridfs.FS
	bid   int64
}

func (biz *deployService) OpenMinion(ctx context.Context, req *param.DeployMinionDownload) (gridfs.File, error) {
	brokerID := req.BrokerID
	if brokerID == 0 {
		brokerID = biz.bid
	}

	// 查询 broker 节点信息
	brkTbl := query.Broker
	brk, err := brkTbl.WithContext(ctx).Where(brkTbl.ID.Eq(brokerID)).First()
	if err != nil {
		return nil, err
	}

	// 根据输入条件匹配合适版本
	bin, err := biz.matchBinary(ctx, req)
	if err != nil {
		return nil, err
	}

	inf, err := biz.gfs.OpenID(bin.FileID)
	if err != nil {
		return nil, err
	}

	// 构造隐写数据
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

	semver := string(bin.Semver)
	hide := &definition.MHide{
		Servername: brk.Servername,
		Addrs:      addrs,
		Semver:     semver,
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
		Edition:    semver,
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

func (biz *deployService) matchBinary(ctx context.Context, req *param.DeployMinionDownload) (*model.MinionBin, error) {
	tbl := query.MinionBin
	if binID := req.ID; binID != 0 { // 如果显式指定了 id，则按照 ID 匹配。
		bin, err := tbl.WithContext(ctx).Where(tbl.ID.Eq(binID)).First()
		if err != nil {
			return nil, err
		}
		if bin.Deprecated {
			return nil, errors.New("该版本已标记为过期")
		}
		return bin, nil
	}

	conds := []gen.Condition{
		tbl.Deprecated.Is(false), // 标记为过期不能下载
		tbl.Goos.Eq(req.Goos),
		tbl.Arch.Eq(req.Arch),
		tbl.Unstable.Is(req.Unstable), // 是否测试版
	}
	if customized := req.Customized; customized == "" {
		conds = append(conds, field.Or(tbl.Customized.Eq(customized), tbl.Customized.IsNull()))
	} else {
		conds = append(conds, tbl.Customized.Eq(customized))
	}

	if semver := string(req.Version); semver != "" {
		conds = append(conds, tbl.Semver.Eq(semver))
	}

	return tbl.WithContext(ctx).Where(conds...).
		Order(tbl.Weight.Desc(), tbl.Semver.Desc()).
		First()
}
