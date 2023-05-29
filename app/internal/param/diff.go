package param

import (
	"bytes"
	"fmt"
	"time"

	"github.com/vela-ssoc/vela-common-mb/dal/model"
)

type TaskChunk struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Dialect bool   `json:"dialect"`
	Hash    string `json:"hash"`
	Chunk   []byte `json:"chunk"`
}

type TaskDiff struct {
	Removes []int64      `json:"removes"` // 需要删除的配置 ID
	Updates []*TaskChunk `json:"updates"` // 需要更新的配置信息
}

func (td TaskDiff) String() string {
	buf := bytes.NewBufferString("[TaskDiff] ")
	rsz, usz := len(td.Removes), len(td.Updates)
	if rsz == 0 && usz == 0 {
		buf.WriteString("无差异")
		return buf.String()
	}

	if rsz != 0 {
		str := fmt.Sprintf("删除 %d 条配置：%v", rsz, td.Removes)
		buf.WriteString(str)
	}
	if usz != 0 {
		str := fmt.Sprintf("更新 %d 条配置：", usz)
		buf.WriteString(str)
		for _, up := range td.Updates {
			buf.WriteString(up.Name)
		}
	}

	return buf.String()
}

// NotModified 与中心端比对没有差异
func (td TaskDiff) NotModified() bool {
	return len(td.Removes) == 0 && len(td.Updates) == 0
}

type TaskStatus struct {
	ID      int64             `json:"id"`      // 配置 ID 由中心端下发
	Dialect bool              `json:"dialect"` // 是否是私有配置，由中心端下发
	Name    string            `json:"name"`    // 配置名称，由中心端下发
	Status  string            `json:"status"`  // 运行状态
	Hash    string            `json:"hash"`    // 配置哈希（目前是 MD5）
	Uptime  time.Time         `json:"uptime"`  // 配置启动时间
	From    string            `json:"from"`    // 配置来源
	Cause   string            `json:"cause"`   // 错误原因
	Runners model.TaskRunners `json:"runners"` // 任务内部模块运行状态
}

type TaskRunner struct {
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`
}

type TaskReport struct {
	Tasks []*TaskStatus `json:"tasks"`
}

func (tr TaskReport) IDMap() map[int64]*TaskStatus {
	size := len(tr.Tasks)
	hm := make(map[int64]*TaskStatus, size)
	for _, task := range tr.Tasks {
		hm[task.ID] = task
	}
	return hm
}

func (tr TaskReport) ToModels(mid int64, inet string) []*model.MinionTask {
	size := len(tr.Tasks)
	ret := make([]*model.MinionTask, 0, size)
	now := time.Now()

	for _, task := range tr.Tasks {
		uptime := task.Uptime
		if uptime.IsZero() {
			uptime = now
		}
		mt := &model.MinionTask{
			SubstanceID: task.ID,
			MinionID:    mid,
			Inet:        inet,
			Dialect:     task.Dialect,
			Name:        task.Name,
			Status:      task.Status,
			Hash:        task.Hash,
			From:        task.From,
			Uptime:      uptime,
			Failed:      task.Cause != "",
			Cause:       task.Cause,
			Runners:     task.Runners,
			CreatedAt:   now,
		}
		ret = append(ret, mt)
	}

	return ret
}
