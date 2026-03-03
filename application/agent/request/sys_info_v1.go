package request

import (
	"github.com/vela-ssoc/ssoc-common/store/model"
)

type SysInfoV1Report struct {
	HostID      string `json:"host_id"`
	Hostname    string `json:"hostname"`
	Release     string `json:"release"`
	Family      string `json:"family"`
	Uptime      int64  `json:"uptime"`
	BootAt      int64  `json:"boot_at"`
	Virtual     string `json:"virtual"`
	VirtualRole string `json:"virtual_role"`
	ProcNumber  int    `json:"proc_number"`
	MemTotal    int    `json:"mem_total"`
	MemFree     int    `json:"mem_free"`
	SwapTotal   int    `json:"swap_total"`
	SwapFree    int    `json:"swap_free"`
	CPUCore     int    `json:"cpu_core"`
	CPUModel    string `json:"cpu_model"`
	AgentTotal  int    `json:"agent_total"`
	AgentAlloc  int    `json:"agent_alloc"`
	Version     string `json:"version"`
}

func (r SysInfoV1Report) Model() *model.MinionSysInfo {
	return &model.MinionSysInfo{
		Release:       r.Release,
		CPUCore:       r.CPUCore,
		MemTotal:      r.MemTotal,
		MemFree:       r.MemFree,
		SwapTotal:     r.SwapTotal,
		SwapFree:      r.SwapFree,
		HostID:        r.HostID,
		Family:        r.Family,
		Uptime:        r.Uptime,
		BootAt:        r.BootAt,
		Virtual:       r.Virtual,
		VirtualRole:   r.VirtualRole,
		ProcNumber:    r.ProcNumber,
		Hostname:      r.Hostname,
		CPUModel:      r.CPUModel,
		AgentTotal:    r.AgentTotal,
		AgentAlloc:    r.AgentAlloc,
		KernelVersion: r.Version,
	}
}
