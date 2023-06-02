package param

import (
	"time"

	"github.com/vela-ssoc/vela-common-mb/dal/model"
)

type InfoRequest struct {
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

func (ir InfoRequest) Model(minionID int64) *model.SysInfo {
	return &model.SysInfo{
		ID:            minionID,
		Release:       ir.Release,
		CPUCore:       ir.CPUCore,
		MemTotal:      ir.MemTotal,
		MemFree:       ir.MemFree,
		SwapTotal:     ir.SwapTotal,
		SwapFree:      ir.SwapFree,
		HostID:        ir.HostID,
		Family:        ir.Family,
		Uptime:        ir.Uptime,
		BootAt:        ir.BootAt,
		Virtual:       ir.Virtual,
		VirtualRole:   ir.VirtualRole,
		ProcNumber:    ir.ProcNumber,
		Hostname:      ir.Hostname,
		CPUModel:      ir.CPUModel,
		AgentTotal:    ir.AgentTotal,
		AgentAlloc:    ir.AgentAlloc,
		KernelVersion: ir.Version,
		UpdatedAt:     time.Now(),
	}
}

type CollectCPU struct {
	CPU       string  `json:"cpu"`
	User      float64 `json:"user"`
	System    float64 `json:"system"`
	Idle      float64 `json:"idle"`
	Nice      float64 `json:"nice"`
	IOWait    float64 `json:"io_wait"`
	Irq       float64 `json:"irq"`
	SoftIRQ   float64 `json:"soft_irq"`
	Steal     float64 `json:"steal"`
	Guest     float64 `json:"guest"`
	GuestNice float64 `json:"guest_nice"`
}
