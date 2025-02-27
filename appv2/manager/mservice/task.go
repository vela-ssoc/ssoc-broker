package mservice

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/vela-ssoc/vela-broker/appv2/manager/mrequest"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mb/dal/query"
	"gorm.io/gen"
	"gorm.io/gen/field"
)

func NewTask(qry *query.Query, lnk mlink.Linker, log *slog.Logger) *Task {
	return &Task{
		qry:   qry,
		lnk:   lnk,
		log:   log,
		limit: make(map[int64]struct{}, 16),
	}
}

type Task struct {
	qry   *query.Query
	lnk   mlink.Linker
	log   *slog.Logger
	mutex sync.Mutex
	limit map[int64]struct{}
}

func (tsk *Task) Push(execID int64) error {
	tsk.mutex.Lock()
	_, exists := tsk.limit[execID]
	if !exists {
		tsk.limit[execID] = struct{}{}
	}
	tsk.mutex.Unlock()
	if exists {
		return errors.New("task already pushing")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	execute := tsk.qry.TaskExecute
	executeDo := execute.WithContext(ctx)

	data, err := executeDo.Where(execute.ID.Eq(execID)).First()
	if err != nil {
		return err
	}

	tsk.execute(data)

	return nil
}

func (tsk *Task) execute(data *model.TaskExecute) error {
	execID := data.ID
	defer func() {
		tsk.mutex.Lock()
		delete(tsk.limit, execID)
		tsk.mutex.Unlock()
	}()

	taskExecuteItem := tsk.qry.TaskExecuteItem
	taskExecuteItemDo := taskExecuteItem.WithContext(context.Background())

	brokerID := tsk.lnk.Link().Ident().ID

	wheres := []gen.Condition{
		taskExecuteItem.BrokerID.Eq(brokerID),
		taskExecuteItem.ExecID.Eq(execID),
		taskExecuteItem.BrokerStatus.IsNull(),
	}
	var buf []*model.TaskExecuteItem
	err := taskExecuteItemDo.Where(wheres...).FindInBatches(&buf, 100, func(tx gen.Dao, batch int) error {
		for _, item := range buf {
			now := time.Now()
			err := tsk.pushTask(item.MinionID, execID, data)

			var updates []field.AssignExpr
			status := &model.TaskStepStatus{Succeed: err == nil, ExecutedAt: now}
			if err != nil {
				status.Reason = err.Error()
				updates = append(updates,
					taskExecuteItem.Finished.Value(true),
					taskExecuteItem.ErrorCode.Value(model.TaskExecuteErrorCodeAgent),
				)
			}
			updates = append(updates, taskExecuteItem.BrokerStatus.Value(status))

			_, _ = tx.Where(taskExecuteItem.ID.Eq(item.ID)).UpdateSimple(updates...)
		}
		return nil
	})

	return err
}

func (tsk *Task) pushTask(minionID, execID int64, data *model.TaskExecute) error {
	req := &mrequest.TaskPushData{
		ID:       data.TaskID,
		ExecID:   execID,
		Name:     data.Name,
		Intro:    data.Intro,
		Code:     data.Code,
		CodeSHA1: data.CodeSHA1,
		Timeout:  data.Timeout.Duration(),
	}

	const rawURL = "/api/v1/agent/task/push"
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	return tsk.lnk.Oneway(ctx, minionID, rawURL, req)
}
