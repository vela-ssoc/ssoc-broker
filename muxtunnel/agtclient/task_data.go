package agtclient

import "github.com/vela-ssoc/ssoc-common-mb/dal/model"

type TaskReportResponse struct {
	Tasks []*model.TaskStatus
}
