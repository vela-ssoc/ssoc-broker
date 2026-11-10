package request

type MinionSysinfo struct {
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

type MinionTag struct {
	Add []string `json:"add" validate:"lte=50,unique,dive,tag"`
	Del []string `json:"del" validate:"lte=50,unique,dive,tag"`
}

func (mt MinionTag) IsZero() bool {
	return len(mt.Del) == 0 && len(mt.Add) == 0
}
