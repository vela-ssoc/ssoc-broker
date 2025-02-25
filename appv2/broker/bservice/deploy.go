package bservice

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/vela-ssoc/vela-broker/appv2/broker/brequest"
	"github.com/vela-ssoc/vela-common-mb/dal/gridfs2"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mba/definition"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

func NewDeploy(qry *query.Query) *Deploy {
	return &Deploy{
		qry: qry,
	}
}

type Deploy struct {
	gfs gridfs2.FS
	qry *query.Query
	log *slog.Logger
}

// Minion 下载
func (dep *Deploy) Minion(ctx context.Context, args *brequest.DeployArguments) (io.ReadCloser, error) {
	attrs := []any{slog.Any("args", args)}
	minionBin, err := dep.findMinionBinary(ctx, args)
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		dep.log.ErrorContext(ctx, "寻找合适的agent二进制出错", attrs...)
		return nil, err
	}

	brokerTbl := dep.qry.Broker
	brokerDo := brokerTbl.WithContext(ctx)
	broker, err := brokerDo.Where(brokerTbl.ID.Eq(args.BrokerID)).First()
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
		dep.log.ErrorContext(ctx, "查找broker节点错误", attrs...)
		return nil, err
	}

	semver := string(minionBin.Semver)
	hide := &definition.MHide{
		Servername: broker.Servername,
		Addrs:      broker.Addresses(),
		Semver:     semver,
		Hash:       minionBin.Hash,
		Size:       minionBin.Size,
		Tags:       args.Tags,
		Goos:       minionBin.Goos,
		Arch:       minionBin.Arch,
		Unload:     args.Unload,
		Unstable:   args.Unstable,
		Customized: args.Customized,
		DownloadAt: time.Now(),
		VIP:        broker.VIP,
		LAN:        broker.LAN,
		Edition:    semver,
	}

	return nil, nil
}

// Script 下载部署安装脚本。
func (dep *Deploy) Script(ctx context.Context, args *brequest.DeployArguments) error {
	// ID         int64        `json:"id"         query:"id"`                            // 客户端安装包 ID
	//	BrokerID   int64        `json:"broker_id"  query:"broker_id" validate:"required"` // 中心端 ID
	//	Goos       string       `json:"goos"       query:"goos"      validate:"omitempty,oneof=linux windows darwin"`
	//	Arch       string       `json:"arch"       query:"arch"      validate:"omitempty,oneof=amd64 386 arm64 arm"`
	//	Version    model.Semver `json:"version"    query:"version"   validate:"omitempty,semver"`
	//	Unload     bool         `json:"unload"     query:"unload"`     // 静默模式
	//	Unstable   bool         `json:"unstable"   query:"unstable"`   // 测试版
	//	Customized string       `json:"customized" query:"customized"` // 定制版标记
	//	Tags       []string     `json:"tags"       query:"tags"      validate:"lte=16,unique,dive,tag"`

	return nil
}

func (dep *Deploy) findMinionBinary(ctx context.Context, args *brequest.DeployArguments) (*model.MinionBin, error) {
	tbl := dep.qry.MinionBin
	tblDo := tbl.WithContext(ctx)
	if id := args.ID; id > 0 { // 直接显式指明 ID
		return tbl.WithContext(ctx).Where(tbl.ID.Eq(id)).First()
	}

	goos, arch := args.Goos, args.Arch
	if goos == "" {
		goos = "linux"
	}
	if arch == "" {
		arch = "amd64"
	}

	wheres := []gen.Condition{field.Or(tbl.Deprecated.IsNull(), tbl.Deprecated.Is(false))}
	wheres = append(wheres, tbl.Goos.Eq(goos), tbl.Arch.Eq(arch))
	if semver := args.Version; semver != "" {
		wheres = append(wheres, tbl.Semver.Eq(string(semver)))
	}
	if args.Unstable {
		wheres = append(wheres, tbl.Unstable.Is(true))
	} else {
		wheres = append(wheres, field.Or(tbl.Unstable.Is(false), tbl.Unstable.IsNull()))
	}
	if customized := args.Customized; customized == "" {
		wheres = append(wheres, field.Or(tbl.Customized.Eq(customized), tbl.Customized.IsNull()))
	} else {
		wheres = append(wheres, tbl.Customized.Eq(customized))
	}

	return tblDo.Where(wheres...).First()
}
