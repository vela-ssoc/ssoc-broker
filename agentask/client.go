package agentask

import (
	"context"
	"errors"
	"log/slog"

	"github.com/vela-ssoc/ssoc-broker/muxtunnel/apiclient"
	model2 "github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/model"
	"github.com/vela-ssoc/ssoc-common-mb/datalayer/polydb"
	"github.com/vela-ssoc/ssoc-common-mb/muxproto"
)

type Client struct {
	cli apiclient.Client
	pdb polydb.Sourcer
	log *slog.Logger
}

func NewClient(cli apiclient.Client, pdb polydb.Sourcer, log *slog.Logger) Client {
	return Client{cli: cli, pdb: pdb, log: log}
}

func (c Client) APIClient() apiclient.Client {
	return c.cli
}

func (c Client) PushStartup(ctx context.Context, mid int64) error {
	data := c.findStartup(ctx, mid)
	host := muxproto.Int64ToHost(mid)
	cli := c.cli.Agent(host)

	args := []any{"minion_id", mid}
	if err := cli.Startup().Push(ctx, data); err != nil {
		args = append(args, "err", err)
		c.log.Error("startup 脚本推送出错", args...)
		return err
	}
	c.log.Info("startup 脚本推送成功", args...)

	return nil
}

func (c Client) ReloadTask(ctx context.Context, mid, subID int64) error {
	// 检查探针节点是否静默
	// 检查配置是否和该节点关联
	args := []any{"minion_id", mid, "substance_id", subID}
	qry := c.pdb.RDB()
	mtbl := qry.Minion
	mdao := mtbl.WithContext(ctx)
	mon, err := mdao.Where(mtbl.ID.Eq(mid)).First()
	if err != nil {
		args = append(args, "err", err)
		c.log.Error("探针节点查询出错", args...)
		return err
	}
	if mon.Unload {
		c.log.Warn("探针节点处于静默模式", args...)
		return nil
	}
	args = append(args, "inet", mon.Inet)

	stbl := qry.Substance
	sdao := stbl.WithContext(ctx)
	sub, err := sdao.Where(stbl.ID.Eq(subID)).First()
	if err != nil {
		args = append(args, "err", err)
		c.log.Error("查询配置信息出错", args...)
		return err
	}
	args = append(args, "substance_name", sub.Name)
	if sub.MinionID != 0 && sub.MinionID != mid {
		return errors.New("没有改配置权限")
	}

	// 检查该节点是否已被排除
	{
		tbl := qry.MinionSubstanceExclude
		dao := tbl.WithContext(ctx)
		cnt, _ := dao.Where(tbl.MinionID.Eq(mid), tbl.SubstanceID.Eq(subID)).Count()
		if cnt != 0 {
			c.log.Warn("探针节点已经排除了此配置", args...)
			return nil
		}
	}

	diff := &apiclient.AgentTaskDiff{Updates: []*apiclient.AgentTaskChunk{{
		ID:       subID,
		Name:     sub.Name,
		Dialect:  sub.MinionID == mon.ID,
		Hash:     sub.Hash,
		Chunk:    sub.Chunk,
		Priority: sub.Priority,
	}}}

	host := muxproto.Int64ToHost(mid)
	cli := c.cli.Agent(host).Task()
	status, nerr := cli.Diff(ctx, diff)
	if nerr != nil {
		args = append(args, "err", nerr)
		c.log.Error("推送配置差异出错", args...)
		return nerr
	}

	ttbl := qry.MinionTask
	tdao := ttbl.WithContext(ctx)
	_, _ = tdao.Where(ttbl.MinionID.Eq(mid)).Delete()
	if status != nil {
		tasks := status.ToModels(mid, mon.Inet)
		if len(tasks) != 0 {
			_ = ttbl.WithContext(ctx).Create(tasks...)
		}
	}

	return nil
}

func (c Client) ResyncTasks(ctx context.Context, mid int64, tires int) error {
	tires = max(tires, 1)
	tires = min(tires, 10)

	args := []any{"minion_id", mid}
	qry := c.pdb.RDB()
	tbl := qry.Minion
	dao := tbl.WithContext(ctx)
	mon, err := dao.Where(tbl.ID.Eq(mid)).First()
	if err != nil {
		c.log.Error("查询节点出错出错", args...)
		return err
	}

	inet, unload := mon.Inet, mon.Unload
	args = append(args, "inet", inet, "unload", unload)
	effects, err := c.EffectSubstances(ctx, mid, inet, unload)
	if err != nil {
		args = append(args, "err", err)
		c.log.Error("查询数据库当前节点生效的配置出错", args...)
		return err
	}

	host := muxproto.Int64ToHost(mid)
	cli := c.cli.Agent(host).Task()
	status, err := cli.Status(ctx) // 先查询当前 agent 的配置运行状态
	if err != nil {
		args = append(args, "err", err)
		c.log.Error("查询当前节点配置运行信息出错", args...)
		return err
	}

	var synced bool
	for range tires {
		diff := status.Diff(mid, effects)
		if diff.NotModified() {
			c.log.Info("节点配置已经同步一致", args...)
			synced = true
			break
		}

		nstatus, nerr := cli.Diff(ctx, diff)
		if nerr != nil {
			args = append(args, "err", nerr)
			c.log.Error("推送配置差异出错", args...)
			break
		}
		status = nstatus
	}
	if !synced {
		c.log.Warn("多次同步依然未能达成同步", args...)
	}

	ttbl := qry.MinionTask
	tdao := ttbl.WithContext(ctx)
	_, _ = tdao.Where(ttbl.MinionID.Eq(mid)).Delete()
	if status != nil {
		tasks := status.ToModels(mid, mon.Inet)
		if len(tasks) != 0 {
			_ = ttbl.WithContext(ctx).Create(tasks...)
		}
	}

	return nil
}

func (c Client) findStartup(ctx context.Context, id int64) model.StartupV1 {
	qry := c.pdb.RDB()
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

func (c Client) EffectSubstances(ctx context.Context, mid int64, inet string, unload bool) (model2.Substances, error) {
	if unload {
		return model2.Substances{}, nil
	}

	// 查询该节点的所有标签
	qry := c.pdb.RDB()
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
