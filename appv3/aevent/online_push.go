package aevent

import (
	"context"
	"time"

	model2 "github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type onlinePush struct {
	life *LifeEvent
	node muxproto.Node
}

func (op *onlinePush) Run() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	info := op.node.Info()
	host := op.node.Host()
	agtcli := op.life.cli.Agent(host)

	// TODO 推送 startup 配置
	mid := info.IDV1
	args := []any{"info", info}
	startup := op.currentStartup(ctx, mid)
	if err := agtcli.Startup().Push(ctx, startup); err != nil {
		attrs := append(args, "err", err)
		op.life.log.Warn("推送 startup 数据出错", attrs...)
	}

	// TODO 配置同步
	minion, err := op.currentInfo(ctx, mid)
	if err != nil {
		args = append(args, "err", err)
		op.life.log.Error("查询当前节点信息出错", args...)
		return
	}

	effects, err := op.allEffectSubstances(ctx, minion)
	if err != nil {
		args = append(args, "err", err)
		op.life.log.Error("查询数据库当前节点生效的配置出错", args...)
		return
	}

	taskcli := agtcli.Task()
	status, err := taskcli.Status(ctx) // 先查询当前 agent 的配置运行状态
	if err != nil {
		args = append(args, "err", err)
		op.life.log.Error("查询当前节点配置运行信息出错", args...)
		return
	}

	for range 3 {
		diff := status.Diff(mid, effects)
		if diff.NotModified() {
			op.life.log.Info("节点配置已经同步一致", args...)
			break
		}

		nstatus, nerr := taskcli.Diff(ctx, diff)
		if nerr != nil {
			args = append(args, "err", nerr)
			op.life.log.Error("推送配置差异出错", args...)
			break
		}
		status = nstatus
	}

	// 更新上报状态表
	qry := op.life.pdb.RDB()
	tbl := qry.MinionTask
	dao := tbl.WithContext(ctx)
	_, _ = dao.Where(tbl.MinionID.Eq(mid)).Delete()
	if status != nil {
		tasks := status.ToModels(mid, minion.Inet)
		if len(tasks) != 0 {
			_ = tbl.WithContext(ctx).Create(tasks...)
		}
	}
}

func (op *onlinePush) currentInfo(ctx context.Context, id int64) (*model2.Minion, error) {
	qry := op.life.pdb.RDB()
	tbl := qry.Minion
	dao := tbl.WithContext(ctx)

	return dao.Where(tbl.ID.Eq(id)).First()
}

func (op *onlinePush) currentStartup(ctx context.Context, id int64) model.StartupV1 {
	qry := op.life.pdb.RDB()
	{
		tbl := qry.Startup
		dao := tbl.WithContext(ctx)
		dat, err := dao.Where(tbl.ID.Eq(id)).First()
		if err == nil && dat.Logger != nil {
			lg := dat.Logger
			data := model.StartupV1{
				Logger: model.StartupV1Logger{
					Level:    lg.Level,
					Filename: lg.Filename,
					Console:  lg.Console,
					Format:   lg.Format,
					Caller:   lg.Caller,
					Skip:     lg.Skip,
				},
			}

			return data
		}
	}

	tbl := qry.StartupFallback
	dao := tbl.WithContext(ctx)
	ret, _ := dao.Order(tbl.ID.Desc()).First()
	if ret == nil {
		ret = new(model2.StartupFallback)
	}
	lg := ret.Logger

	return model.StartupV1{
		Logger: model.StartupV1Logger{
			Level:    lg.Level,
			Filename: lg.Filename,
			Console:  lg.Console,
			Format:   lg.Format,
			Caller:   lg.Caller,
			Skip:     lg.Skip,
		},
	}
}

func (op *onlinePush) allEffectSubstances(ctx context.Context, minion *model2.Minion) (model2.Substances, error) {
	if minion.Unload { // 静默模式下，所有的配置都不加载运行。
		return nil, nil
	}

	// 查询该节点的所有标签
	qry := op.life.pdb.RDB()
	mid, inet := minion.ID, minion.Inet
	tags := make([]string, 0, 10)
	tagTbl := qry.MinionTag
	_ = tagTbl.WithContext(ctx).
		Distinct(tagTbl.Tag).
		Where(tagTbl.MinionID.Eq(mid)).
		Scan(&tags)

	// 查询节点的配置
	var subIDs []int64
	if len(tags) != 0 {
		effTbl := qry.Effect
		effects, _ := effTbl.WithContext(ctx).
			Where(effTbl.Tag.In(tags...), effTbl.Enable.Is(true)).
			Find()
		subIDs = model2.Effects(effects).Exclusion(inet)
	}

	subTbl := qry.Substance
	dao := subTbl.WithContext(ctx).
		Where(subTbl.MinionID.Eq(mid))
	if len(subIDs) != 0 {
		dao.Or(subTbl.ID.In(subIDs...))
	}
	subs, err := dao.Find()
	if len(subs) == 0 {
		return subs, err
	}
	// 剔除排除的配置
	excTbl := qry.MinionSubstanceExclude
	excDao := excTbl.WithContext(ctx)
	excludes, _ := excDao.Where(excTbl.MinionID.Eq(mid)).Find()
	subs = model2.MinionSubstanceExcludes(excludes).Filter(subs)
	substances := model2.Substances(subs)
	substances.Sort()

	return substances, nil
}
