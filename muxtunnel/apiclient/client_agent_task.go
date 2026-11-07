package apiclient

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vela-ssoc/ssoc-common-mb/dal/model"
	"github.com/vela-ssoc/ssoc-common-mb/httpclient"
)

type AgentTaskClient struct {
	cli httpclient.Client
	log *slog.Logger
	agt AgentClient
}

// Status 获取 agent 最新的配置状态。
func (atc AgentTaskClient) Status(ctx context.Context) (*AgentTaskStatusOut, error) {
	strURL := atc.agt.buildStringURL("/api/v1/agent/task/status")
	out := new(AgentTaskStatusOut)
	if err := atc.cli.PostJSON(ctx, strURL, nil, out); err != nil {
		return nil, err
	}

	return out, nil
}

// Diff 通知 agent 节点配置差异，agent 执行合并后返回最新的配置状态。
func (atc AgentTaskClient) Diff(ctx context.Context, in *AgentTaskDiff) (*AgentTaskStatusOut, error) {
	strURL := atc.agt.buildStringURL("/api/v1/agent/task/diff")
	out := new(AgentTaskStatusOut)
	if err := atc.cli.PostJSON(ctx, strURL, in, out); err != nil {
		return nil, err
	}

	return out, nil
}

type AgentTaskStatusOut struct {
	Tasks []*AgentTaskStatusItem `json:"tasks"`
}

type AgentTaskStatusItem struct {
	ID      int64             `json:"id"`      // 配置 ID 由中心端下发
	Dialect bool              `json:"dialect"` // 是否是私有配置，由中心端下发
	Name    string            `json:"name"`    // 配置名称，由中心端下发
	Status  string            `json:"status"`  // 运行状态
	Hash    string            `json:"hash"`    // 配置哈希（目前是 MD5）
	Uptime  time.Time         `json:"uptime"`  // 配置启动时间
	From    string            `json:"from"`    // 配置来源，ssoc 平台下发的叫 tunnel（重要）
	Link    string            `json:"link"`    // 外链，agent 根据代码分析而得。
	Cause   string            `json:"cause"`   // 错误原因
	Runners model.TaskRunners `json:"runners"` // 任务内部模块运行状态
}

func (tr AgentTaskStatusOut) IDMap() map[int64]*AgentTaskStatusItem {
	size := len(tr.Tasks)
	hm := make(map[int64]*AgentTaskStatusItem, size)
	for _, task := range tr.Tasks {
		hm[task.ID] = task
	}
	return hm
}

func (tr AgentTaskStatusOut) ToModels(mid int64, inet string) []*model.MinionTask {
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
			Link:        task.Link,
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

func (tr AgentTaskStatusOut) Diff(mid int64, subs model.Substances) *AgentTaskDiff {
	hm := tr.IDMap()
	removes := make([]int64, 0, 10)
	updates := make([]*AgentTaskChunk, 0, 10)

	for _, sub := range subs {
		id := sub.ID
		rp, ok := hm[id]
		delete(hm, id)
		if ok && rp.Hash == sub.Hash {
			continue
		}

		chk := &AgentTaskChunk{
			ID:       id,
			Name:     sub.Name,
			Dialect:  sub.MinionID == mid,
			Hash:     sub.Hash,
			Chunk:    sub.Chunk,
			Priority: sub.Priority,
		}
		updates = append(updates, chk)
	}

	for _, rp := range hm {
		if rp.From == "tunnel" {
			removes = append(removes, rp.ID)
		}
	}

	diff := &AgentTaskDiff{
		Removes: removes,
		Updates: updates,
	}

	return diff
}

type AgentTaskDiff struct {
	Removes []int64           `json:"removes"` // 需要删除的配置 ID
	Updates []*AgentTaskChunk `json:"updates"` // 需要更新的配置信息
}

type AgentTaskChunk struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Dialect  bool   `json:"dialect"`
	Hash     string `json:"hash"`
	Chunk    []byte `json:"chunk"`
	Priority int64  `json:"priority"`
}

// NotModified 与中心端比对没有差异
func (td AgentTaskDiff) NotModified() bool {
	return len(td.Removes) == 0 && len(td.Updates) == 0
}

func (td AgentTaskDiff) String() string {
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
			buf.WriteString(", ")
		}
	}

	return buf.String()
}
