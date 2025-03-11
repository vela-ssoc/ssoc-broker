package mservice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"gorm.io/gorm"

	"github.com/vela-ssoc/vela-broker/bridge/telecom"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mba/ciphertext"
	"gorm.io/gen"
)

func NewSystem(link telecom.Linker, qry *query.Query, gfs gridfs.FS, log *slog.Logger) *System {
	return &System{
		link: link,
		qry:  qry,
		gfs:  gfs,
		log:  log,
	}
}

type System struct {
	link   telecom.Linker
	qry    *query.Query
	gfs    gridfs.FS
	log    *slog.Logger
	exit   atomic.Bool
	update atomic.Bool
}

func (sys *System) Exit() {
	if !sys.exit.CompareAndSwap(false, true) {
		sys.log.Warn("收到重复的退出命令")
		return
	}
	defer sys.exit.Store(false)

	sys.log.Warn("程序准备退出")

	time.Sleep(100 * time.Millisecond)
	os.Exit(0)
}

func (sys *System) Update(semver model.Semver) error {
	ident := sys.link.Ident()
	goos, arch := ident.Goos, ident.Arch
	attrs := []any{slog.Any("goos", goos), slog.Any("arch", arch)}

	if !sys.update.CompareAndSwap(false, true) {
		sys.log.Warn("收到重复的升级命令", attrs...)
	}
	defer sys.update.Store(true)

	sys.log.Info("开始检查更新", attrs...)

	currentVersion := model.Semver(ident.Semver)
	currentVersionNum := currentVersion.Uint64()
	if semver != "" {
		attrs = append(attrs, slog.Any("update_version", semver))
		attrs = append(attrs, slog.Any("current_version", currentVersion))
		if semver == currentVersion {
			sys.log.Warn("目标版本一致，无需升级", attrs...)
			return nil
		}

		updateVersionNum := semver.Uint64()
		if updateVersionNum < currentVersionNum {
			sys.log.Warn("更新版本低于当前运行版本，请注意降级后的程序兼容性", attrs...)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 获取版本
	tbl := sys.qry.BrokerBin
	wheres := []gen.Condition{tbl.Goos.Eq(goos), tbl.Arch.Eq(arch)}
	if semver != "" {
		wheres = append(wheres, tbl.Semver.Eq(semver.String()))
	} else {
		wheres = append(wheres, tbl.SemverWeight.Gt(currentVersionNum))
	}
	brokerBin, err := tbl.WithContext(ctx).
		Where(wheres...).
		Order(tbl.SemverWeight.Desc()).
		First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			sys.log.Warn("没有合适的更新包", attrs...)
		} else {
			attrs = append(attrs, slog.Any("error", err))
			sys.log.Error("查询升级包错误", attrs...)
		}
		return err
	}
	attrs = append(attrs, slog.Any("target_semver", brokerBin.Semver))
	sys.log.Info("已找到升级包文件", attrs...)

	hide := sys.link.Hide()
	hide.Semver = brokerBin.Semver.String()

	enc, exx := ciphertext.EncryptPayload(hide)
	if exx != nil {
		return exx
	}

	gf, err := sys.gfs.OpenID(brokerBin.FileID)
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		sys.log.Error("打开 gridfs 文件出错", attrs...)
		return err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer gf.Close()

	file := gridfs.Merge(gf, enc)
	exeName, err := sys.saveFile(brokerBin, file)
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		sys.log.Error("文件保存到磁盘出错", attrs...)
		return err
	}

	const linkname = "ssoc-broker"
	attrs = append(attrs, slog.String("linkname", linkname))
	// 先删除已存在的软链接。
	if err = os.Remove(linkname); err != nil && !os.IsNotExist(err) {
		attrs = append(attrs, slog.Any("error", err))
		sys.log.Error("删除软链接出错", attrs...)
		return err
	}

	if err = os.Symlink(exeName, linkname); err != nil {
		attrs = append(attrs, slog.Any("error", err))
		sys.log.Error("创建软链接出错", attrs...)
	}
	time.Sleep(300 * time.Millisecond)
	os.Exit(0)

	return nil
}

func (sys *System) saveFile(bin *model.BrokerBin, r io.Reader) (string, error) {
	name := bin.Name
	f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		if os.IsExist(err) {
			name = bin.Name + "-" + strconv.FormatInt(bin.ID, 10)
			_ = os.Remove(name)
			f, err = os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o755)
		}
	}
	if err != nil {
		return "", err
	}
	//goland:noinspection GoUnhandledErrorResult
	defer f.Close()
	_, err = io.Copy(f, r)

	return name, err
}
