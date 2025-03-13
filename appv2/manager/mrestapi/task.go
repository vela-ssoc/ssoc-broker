package mrestapi

import (
	"github.com/vela-ssoc/ssoc-broker/appv2/manager/mrequest"
	"github.com/vela-ssoc/ssoc-broker/appv2/manager/mservice"
	"github.com/xgfone/ship/v5"
)

func NewTask(svc *mservice.Task) *Task {
	return &Task{svc: svc}
}

type Task struct {
	svc *mservice.Task
}

func (tsk *Task) BindRoute(r *ship.RouteGroupBuilder) error {
	r.Route("/task/push").POST(tsk.Push)
	return nil
}

func (tsk *Task) Route(r *ship.RouteGroupBuilder) {
	r.Route("/task/push").POST(tsk.Push)
}

func (tsk *Task) Push(c *ship.Context) error {
	req := new(mrequest.TaskPush)
	if err := c.Bind(req); err != nil {
		return err
	}

	tsk.svc.Push(req.ExecID)

	return nil
}
