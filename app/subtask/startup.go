package subtask

import (
	"context"
	"encoding/json"
	"time"

	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/vela-ssoc/vela-common-mb/gopool"
	"github.com/vela-ssoc/vela-common-mb/logback"
	"github.com/vela-ssoc/vela-common-mba/definition"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

func Startup(lnk mlink.Huber, id int64, slog logback.Logger) gopool.Runner {
	return &startupTask{
		id:      id,
		slog:    slog,
		cycle:   5,
		timeout: 10 * time.Second,
		lnk:     lnk,
	}
}

type startupTask struct {
	id      int64
	slog    logback.Logger
	cycle   int
	timeout time.Duration
	lnk     mlink.Huber
}

func (st *startupTask) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), st.timeout)
	defer cancel()

	dat, err := st.find(ctx)
	if err != nil {
		st.slog.Warnf("查询 startup 配置出错：%s", err)
		return
	}

	tbl := query.Startup
	var assigns []field.AssignExpr
	path := "/api/v1/agent/startup"
	if err = st.lnk.Oneway(st.id, path, dat); err != nil {
		assigns = append(assigns, tbl.Failed.Value(true), tbl.Reason.Value(err.Error()))
		st.slog.Warnf("推送 startup 失败：%s", err)
	} else {
		st.slog.Info("推送 startup 成功")
		assigns = append(assigns, tbl.Failed.Value(false), tbl.Reason.Value(""))
	}
	if _, err = tbl.WithContext(ctx).
		Where(tbl.ID.Eq(st.id)).
		UpdateColumnSimple(assigns...); err != nil {
		st.slog.Warnf("startup 执行结果写表失败：%s", err)
	}
}

func (st *startupTask) find(ctx context.Context) (*definition.Startup, error) {
	tbl := query.Startup
	dat, err := tbl.WithContext(ctx).Where(tbl.ID.Eq(st.id)).First()
	if err == nil {
		ret := st.convert(dat)
		return ret, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 查询通用配置
	id := "global.startup.param"
	stTbl := query.Store
	sto, err := stTbl.WithContext(ctx).Where(stTbl.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	var ret definition.Startup
	err = json.Unmarshal(sto.Value, &ret)

	return &ret, err
}

func (st *startupTask) convert(dat *model.Startup) *definition.Startup {
	node := dat.Node
	cons := dat.Console
	logg := dat.Logger

	exts := make([]*definition.StartupExtend, 0, 8)
	for _, e := range dat.Extends {
		exts = append(exts, &definition.StartupExtend{
			Name:  e.Name,
			Type:  e.Type,
			Value: e.Value,
		})
	}

	return &definition.Startup{
		Node: definition.StartupNode{DNS: node.DNS, Prefix: node.Prefix},
		Logger: definition.StartupLogger{
			Level:    logg.Level,
			Filename: logg.Filename,
			Console:  logg.Console,
			Format:   logg.Format,
			Caller:   logg.Caller,
			Skip:     logg.Skip,
		},
		Console: definition.StartupConsole{
			Enable:  cons.Enable,
			Network: cons.Network,
			Address: cons.Address,
			Script:  cons.Script,
		},
		Extends: exts,
	}
}
