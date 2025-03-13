package agtapi

import (
	"encoding/json"
	"time"

	"github.com/vela-ssoc/ssoc-broker/app/internal/param"
	"github.com/vela-ssoc/ssoc-broker/app/route"
	"github.com/vela-ssoc/ssoc-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"github.com/xgfone/ship/v5"
	"gorm.io/gen"
	"gorm.io/gen/field"
	"gorm.io/gorm/clause"
)

func Task(qry *query.Query) route.Router {
	return &taskREST{qry: qry}
}

type taskREST struct {
	qry *query.Query
}

func (rest *taskREST) Route(r *ship.RouteGroupBuilder) {
	r.Route("/broker/task/status").
		Data(route.Named("上报任务运行状态")).POST(rest.Status)
	r.Route("/broker/task/report").
		Data(route.Named("上报任务运行状态(new)")).POST(rest.Report)
}

func (rest *taskREST) Status(c *ship.Context) error {
	var req param.TaskReport
	if err := c.Bind(&req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)

	// 删除旧数据
	mid := inf.Issue().ID
	inet := inf.Inet().String()
	dats := req.ToModels(mid, inet)

	tbl := rest.qry.MinionTask
	_, _ = tbl.WithContext(ctx).Where(tbl.MinionID.Eq(mid)).Delete()
	if len(dats) != 0 {
		_ = tbl.WithContext(ctx).Clauses(clause.OnConflict{UpdateAll: true}).Save(dats...)
	}

	return nil
}

func (rest *taskREST) Report(c *ship.Context) error {
	req := new(ReportData)
	if err := c.Bind(req); err != nil {
		return err
	}

	ctx := c.Request().Context()
	inf := mlink.Ctx(ctx)
	mid := inf.Issue().ID

	tbl := rest.qry.TaskExecuteItem

	wheres := []gen.Condition{
		tbl.ExecID.Eq(req.ExecID),
		tbl.TaskID.Eq(req.ID),
		tbl.MinionID.Eq(mid),
	}

	succeed := req.Succeed
	status := &model.TaskStepStatus{
		Succeed:    succeed,
		Reason:     req.Reason,
		ExecutedAt: time.Now(),
	}
	updates := []field.AssignExpr{
		tbl.Finished.Value(true),
		tbl.Succeed.Value(succeed),
		tbl.Result.Value(req.Result),
		tbl.MinionStatus.Value(status),
	}
	if succeed {
		updates = append(updates, tbl.ErrorCode.Value(model.TaskExecuteErrorCodeSucceed))
	} else {
		updates = append(updates, tbl.ErrorCode.Value(model.TaskExecuteErrorCodeExec))
	}
	_, err := tbl.WithContext(ctx).
		Where(wheres...).
		UpdateSimple(updates...)

	return err
}

type ReportData struct {
	ID      int64           `json:"id"`
	ExecID  int64           `json:"exec_id"`
	Succeed bool            `json:"succeed"`
	Reason  string          `json:"reason"`
	Result  json.RawMessage `json:"result"`
}
