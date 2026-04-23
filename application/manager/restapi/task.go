package restapi

import (
	"github.com/vela-ssoc/ssoc-broker/application/manager/request"
	"github.com/vela-ssoc/ssoc-broker/application/manager/service"
	"github.com/xgfone/ship/v5"
)

func NewTask(svc *service.Task) *Task {
	return &Task{svc: svc}
}

type Task struct {
	svc *service.Task
}

func (tsk *Task) BindRoute(r *ship.RouteGroupBuilder) error {
	r.Route("/task/push").POST(tsk.Push)
	return nil
}

func (tsk *Task) Route(r *ship.RouteGroupBuilder) {
	r.Route("/task/push").POST(tsk.Push)
}

func (tsk *Task) Push(c *ship.Context) error {
	req := new(request.TaskPush)
	if err := c.Bind(req); err != nil {
		return err
	}

	tsk.svc.Push(req.ExecID)

	return nil
}
