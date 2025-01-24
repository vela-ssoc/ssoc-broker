package agtapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"time"

	"github.com/vela-ssoc/vela-broker/app/agtsvc"
	"github.com/vela-ssoc/vela-broker/app/internal/param"
	"github.com/vela-ssoc/vela-broker/app/route"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/xgfone/ship/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Collect(qry *query.Query, svc agtsvc.CollectService) route.Router {
	return &collectREST{
		qry: qry,
		svc: svc,
	}
}

type collectREST struct {
	qry *query.Query
	svc agtsvc.CollectService
}

func (rest *collectREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/collect/agent/sysinfo").POST(rest.Sysinfo)
	r.Route("/broker/collect/agent/process").POST(rest.ProcessDiff)
	r.Route("/broker/collect/agent/process/diff").POST(rest.ProcessDiff)
	r.Route("/broker/collect/agent/process/full").POST(rest.ProcessFull)
	r.Route("/broker/collect/agent/process/sync").POST(rest.ProcessSync)
	r.Route("/broker/collect/agent/logon").POST(rest.Logon)
	r.Route("/broker/collect/agent/listen").POST(rest.ListenDiff)
	r.Route("/broker/collect/agent/listen/diff").POST(rest.ListenDiff)
	r.Route("/broker/collect/agent/listen/full").POST(rest.ListenFull)
	r.Route("/broker/collect/agent/account").POST(rest.AccountDiff)
	r.Route("/broker/collect/agent/account/diff").POST(rest.AccountDiff)
	r.Route("/broker/collect/agent/account/full").POST(rest.AccountFull)
	r.Route("/broker/collect/agent/group").POST(rest.GroupDiff)
	r.Route("/broker/collect/agent/group/diff").POST(rest.GroupDiff)
	r.Route("/broker/collect/agent/group/full").POST(rest.GroupFull)
	r.Route("/broker/collect/agent/sbom").POST(rest.Sbom)
	r.Route("/broker/collect/agent/cpu").POST(rest.CPU)
}

func (rest *collectREST) Sysinfo(c *ship.Context) error {
	var req param.InfoRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	dat := req.Model(inf.Issue().ID)

	return rest.svc.Sysinfo(dat)
}

func (rest *collectREST) ProcessDiff(c *ship.Context) error {
	var req param.CollectProcessDiff
	if err := c.Bind(&req); err != nil {
		return err
	}

	tbl := rest.qry.MinionProcess
	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	pids := req.Deletes
	for _, p := range req.Updates {
		pids = append(pids, p.Pid)
	}
	if len(pids) != 0 {
		_, _ = tbl.WithContext(ctx).
			Where(tbl.MinionID.Eq(mid), tbl.Pid.In(pids...)).
			Delete()
	}

	dats := make([]*model.MinionProcess, 0, 32)
	for _, p := range req.Creates {
		proc := p.Model(mid, inet)
		dats = append(dats, proc)
	}
	for _, p := range req.Updates {
		proc := p.Model(mid, inet)
		dats = append(dats, proc)
	}

	if err := tbl.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(dats, 100); err != nil {
		c.Warnf("进程差异信息插入错误：%s", err)
		return err
	}

	return nil
}

func (rest *collectREST) ProcessFull(c *ship.Context) error {
	r := c.Request()
	var req []*param.CollectProcess
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	tbl := rest.qry.MinionProcess
	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	// 1. 删除所有的该节点数据
	size := len(req)
	_, err := tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()
	if err != nil || size == 0 {
		return err
	}

	dats := make([]*model.MinionProcess, 0, size)
	for _, p := range req {
		proc := p.Model(mid, inet)
		dats = append(dats, proc)
	}

	if err = tbl.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(dats, 100); err != nil {
		c.Warnf("进程全量信息插入错误：%s", err)
	}

	return nil
}

func (rest *collectREST) Logon(c *ship.Context) error {
	var req param.CollectLogonRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()
	dat := req.Model(mid, inet)
	err := rest.qry.MinionLogon.WithContext(ctx).Create(dat)

	return err
}

func (rest *collectREST) ListenDiff(c *ship.Context) error {
	var req param.CollectListenDiff
	if err := c.Bind(&req); err != nil {
		return err
	}

	tbl := rest.qry.MinionListen
	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	rids := req.Deletes
	for _, p := range req.Updates {
		rids = append(rids, p.RecordID)
	}
	if len(rids) != 0 {
		_, _ = tbl.WithContext(ctx).
			Where(tbl.MinionID.Eq(mid), tbl.RecordID.In(rids...)).
			Delete()
	}

	dats := make([]*model.MinionListen, 0, 32)
	for _, p := range req.Creates {
		lis := p.Model(mid, inet)
		dats = append(dats, lis)
	}
	for _, p := range req.Updates {
		lis := p.Model(mid, inet)
		dats = append(dats, lis)
	}

	if err := tbl.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(dats, 100); err != nil {
		c.Warnf("监听差异信息插入错误：%s", err)
		return err
	}

	return nil
}

func (rest *collectREST) ListenFull(c *ship.Context) error {
	var req []*param.CollectListenItem
	r := c.Request()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	// 1. 删除所有的该节点数据
	size := len(req)
	tbl := rest.qry.MinionListen
	_, err := tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()
	if err != nil || size == 0 {
		return err
	}

	dats := make([]*model.MinionListen, 0, size)
	for _, p := range req {
		lis := p.Model(mid, inet)
		dats = append(dats, lis)
	}

	if err = tbl.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(dats, 100); err != nil {
		c.Warnf("监听全量信息插入错误：%s", err)
	}

	return err
}

func (rest *collectREST) AccountDiff(c *ship.Context) error {
	var req param.CollectAccountDiff
	if err := c.Bind(&req); err != nil {
		return err
	}

	tbl := rest.qry.MinionAccount
	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	names := req.Deletes
	for _, a := range req.Updates {
		names = append(names, a.Name)
	}
	if len(names) != 0 {
		_, _ = tbl.WithContext(ctx).
			Where(tbl.MinionID.Eq(mid), tbl.Name.In(names...)).
			Delete()
	}

	dats := make([]*model.MinionAccount, 0, 32)
	for _, p := range req.Creates {
		acc := p.Model(mid, inet)
		dats = append(dats, acc)
	}
	for _, p := range req.Updates {
		acc := p.Model(mid, inet)
		dats = append(dats, acc)
	}

	if err := tbl.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(dats, 100); err != nil {
		c.Warnf("账户差异信息插入错误：%s", err)
		return err
	}

	return nil
}

func (rest *collectREST) AccountFull(c *ship.Context) error {
	var req []*param.CollectAccountItem
	r := c.Request()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	// 1. 删除所有的该节点数据
	size := len(req)
	dats := make([]*model.MinionAccount, 0, size)
	for _, p := range req {
		acc := p.Model(mid, inet)
		dats = append(dats, acc)
	}

	return rest.svc.AccountFull(mid, dats)
}

func (rest *collectREST) GroupDiff(c *ship.Context) error {
	var req param.CollectGroupDiff
	if err := c.Bind(&req); err != nil {
		return err
	}

	tbl := rest.qry.MinionGroup
	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	names := req.Deletes
	for _, a := range req.Updates {
		names = append(names, a.Name)
	}
	if len(names) != 0 {
		_, _ = tbl.WithContext(ctx).
			Where(tbl.MinionID.Eq(mid), tbl.Name.In(names...)).
			Delete()
	}

	dats := make([]*model.MinionGroup, 0, 32)
	for _, p := range req.Creates {
		g := p.Model(mid, inet)
		dats = append(dats, g)
	}
	for _, p := range req.Updates {
		g := p.Model(mid, inet)
		dats = append(dats, g)
	}

	if err := tbl.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(dats, 100); err != nil {
		c.Warnf("用户组差异信息插入错误：%s", err)
		return err
	}

	return nil
}

func (rest *collectREST) GroupFull(c *ship.Context) error {
	var req []*param.CollectGroupItem
	r := c.Request()
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	// 1. 删除所有的该节点数据
	size := len(req)
	tbl := rest.qry.MinionGroup
	_, err := tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()
	if err != nil || size == 0 {
		return err
	}

	dats := make([]*model.MinionGroup, 0, size)
	for _, p := range req {
		g := p.Model(mid, inet)
		dats = append(dats, g)
	}

	if err = tbl.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(dats, 100); err != nil {
		c.Warnf("用户组全量信息插入错误：%s", err)
	}

	return err
}

func (rest *collectREST) Sbom(c *ship.Context) error {
	var req param.SbomRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid, inet := inf.Issue().ID, inf.Inet().String()

	now := time.Now()
	if req.ModifyAt.IsZero() {
		req.ModifyAt = now
	}
	req.Filename = filepath.Clean(req.Filename)

	// 查询数据
	ptjTbl := rest.qry.SBOMProject
	old, err := ptjTbl.WithContext(ctx).
		Where(ptjTbl.MinionID.Eq(mid), ptjTbl.Filepath.Eq(req.Filename)).
		First()
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			c.Warnf("查询节点 %d %s 旧的 SBOM 错误：%s", mid, req.Filename, err)
			return err
		}
		return rest.sbomInsert(ctx, mid, inet, &req)
	}

	if old.SHA1 == req.Checksum { // 哈希不变无需更新
		return nil
	}

	// 有变化就删除后插入
	_, _ = ptjTbl.WithContext(ctx).Where(ptjTbl.ID.Eq(old.ID)).Delete()
	comTbl := rest.qry.SBOMComponent
	_, _ = comTbl.WithContext(ctx).Where(comTbl.ProjectID.Eq(old.ID)).Delete()

	return rest.sbomInsert(ctx, mid, inet, &req)
}

func (rest *collectREST) CPU(c *ship.Context) error {
	var req param.CollectCPU
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	c.Infof("收到 %s CPU 上报信息", inf.Inet())

	return nil
}

func (rest *collectREST) sbomInsert(ctx context.Context, minionID int64, inet string, r *param.SbomRequest) error {
	pjt := &model.SBOMProject{
		MinionID:     minionID,
		Inet:         inet,
		Filepath:     r.Filename,
		SHA1:         r.Checksum,
		Size:         int(r.Size),
		ComponentNum: len(r.SDKs),
		PID:          r.Process.PID,
		Exe:          r.Process.Exe,
		Username:     r.Process.Username,
		ModifyAt:     r.ModifyAt,
	}

	if err := rest.qry.SBOMProject.WithContext(ctx).
		Create(pjt); err != nil {
		return err
	}

	components := r.Components(minionID, inet, pjt.ID)
	if len(components) == 0 {
		return nil
	}

	return rest.qry.SBOMComponent.WithContext(ctx).
		CreateInBatches(components, 100)
}

func (rest *collectREST) ProcessSync(c *ship.Context) error {
	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	var dats param.ProcSimples
	tbl := rest.qry.MinionProcess
	_ = tbl.WithContext(ctx).
		Select(tbl.Name, tbl.State, tbl.Pid, tbl.Pgid, tbl.Ppid, tbl.Cmdline,
			tbl.Username, tbl.Cwd, tbl.Executable, tbl.Args).
		Where(tbl.MinionID.Eq(inf.Issue().ID)).
		Scan(&dats)
	ret := &param.Data{Data: dats}

	return c.JSON(http.StatusOK, ret)
}
