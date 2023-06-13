package param

import (
	"strings"
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
type CollectProcessDiff struct {
	Creates []*CollectProcess `json:"creates"` // 新增的进程
	Updates []*CollectProcess `json:"updates"` // 更新的进程
	Deletes []int             `json:"deletes"` // 删除的 PID
}

type CollectProcess struct {
	Name         string    `json:"name"`
	State        string    `json:"state"`
	Pid          int       `json:"pid"`
	Ppid         int       `json:"ppid"`
	Pgid         uint32    `json:"pgid"`
	Cmdline      string    `json:"cmdline"`
	Username     string    `json:"username"`
	Cwd          string    `json:"cwd"`
	Executable   string    `json:"executable"` // linux
	Args         []string  `json:"args"`
	UserTicks    uint64    `json:"user_ticks"`
	TotalPct     float64   `json:"total_pct"`
	TotalNormPct float64   `json:"total_norm_pct"`
	SystemTicks  uint64    `json:"system_ticks"`
	TotalTicks   uint64    `json:"total_ticks"`
	StartTime    string    `json:"start_time"`
	MemSize      uint64    `json:"mem_size"`
	RssBytes     uint64    `json:"rss_bytes"`
	RssPct       float64   `json:"rss_pct"`
	Share        uint64    `json:"share"`
	Checksum     string    `json:"checksum"`
	ModifyTime   time.Time `json:"modify_time"`
	CreateTime   time.Time `json:"create_time"`
}

// Model 将 process 转为 model.MinionProcess
func (p CollectProcess) Model(minionID int64, inet string) *model.MinionProcess {
	if p.CreateTime.IsZero() {
		p.CreateTime = time.Now()
	}
	if p.ModifyTime.IsZero() {
		p.ModifyTime = time.Now()
	}

	return &model.MinionProcess{
		MinionID:     minionID,
		Inet:         inet,
		Name:         p.Name,
		State:        p.State,
		Pid:          p.Pid,
		Ppid:         p.Ppid,
		Pgid:         p.Pgid,
		Cmdline:      p.Cmdline,
		Username:     p.Username,
		Cwd:          p.Cwd,
		Executable:   p.Executable,
		Args:         p.Args,
		UserTicks:    p.UserTicks,
		TotalPct:     p.TotalPct,
		TotalNormPct: p.TotalNormPct,
		SystemTicks:  p.SystemTicks,
		TotalTicks:   p.TotalTicks,
		StartTime:    p.StartTime,
		MemSize:      p.MemSize,
		RssBytes:     p.RssBytes,
		RssPct:       p.RssPct,
		Share:        p.Share,
		Checksum:     p.Checksum,
		ModifiedAt:   p.ModifyTime,
		CreatedTime:  p.CreateTime,
	}
}

type CollectLogonRequest struct {
	User    string    `json:"user"  validate:"required,lte=255"`
	Addr    string    `json:"addr"  validate:"omitempty,lte=100"`
	Class   string    `json:"class" validate:"omitempty,lte=255"`
	Time    time.Time `json:"time"`
	Type    string    `json:"type"`
	PID     int       `json:"pid"`
	Device  string    `json:"device"`
	Process string    `json:"process"`
}

func (r CollectLogonRequest) Model(minionID int64, inet string) *model.MinionLogon {
	at := r.Time
	if at.IsZero() {
		at = time.Now()
	}

	return &model.MinionLogon{
		MinionID: minionID,
		Inet:     inet,
		User:     r.User,
		Addr:     r.Addr,
		Msg:      r.Class,
		LogonAt:  at,
		Type:     r.Type,
		PID:      r.PID,
		Device:   r.Device,
		Process:  r.Process,
	}
}

type CollectListenDiff struct {
	Creates []*CollectListenItem `json:"creates"` // 新增的 Listen
	Updates []*CollectListenItem `json:"updates"` // 更新的 Listen
	Deletes []string             `json:"deletes"` // 删除的 Listen RecordID
}

type CollectListenItem struct {
	RecordID  string `json:"record_id"`
	PID       uint32 `json:"pid"`
	FD        int    `json:"fd"`
	Family    uint8  `json:"family"`
	Protocol  uint8  `json:"protocol"`
	LocalIP   string `json:"local_ip"`
	LocalPort int    `json:"local_port"`
	Path      string `json:"path"`
	State     string `json:"state"`
	Process   string `json:"process"`
	Username  string `json:"username"`
}

func (l CollectListenItem) Model(minionID int64, inet string) *model.MinionListen {
	return &model.MinionListen{
		MinionID:  minionID,
		Inet:      inet,
		RecordID:  l.RecordID,
		PID:       l.PID,
		FD:        l.FD,
		Family:    l.Family,
		Protocol:  l.Protocol,
		LocalIP:   l.LocalIP,
		LocalPort: l.LocalPort,
		Path:      l.Path,
		Process:   l.Process,
		Username:  l.Username,
	}
}

type CollectGroupItem struct {
	Name        string `json:"name"`
	GID         string `json:"gid"`
	Description string `json:"description"`
}

type CollectGroupDiff struct {
	Creates []*CollectGroupItem `json:"creates"` // 新增的账户
	Updates []*CollectGroupItem `json:"updates"` // 更新的账户
	Deletes []string            `json:"deletes"` // 删除的账户名
}

func (g CollectGroupItem) Model(minionID int64, inet string) *model.MinionGroup {
	return &model.MinionGroup{
		MinionID:    minionID,
		Inet:        inet,
		Name:        g.Name,
		GID:         g.GID,
		Description: g.Description,
	}
}

type CollectAccountDiff struct {
	Creates []*CollectAccountItem `json:"creates"` // 新增的账户
	Updates []*CollectAccountItem `json:"updates"` // 更新的账户
	Deletes []string              `json:"deletes"` // 删除的账户名
}

type CollectAccountItem struct {
	Name        string `json:"name"        gorm:"column:name"`
	LoginName   string `json:"login_name"  gorm:"column:login_name"`
	UID         string `json:"uid"         gorm:"column:uid"`
	GID         string `json:"gid"         gorm:"column:gid"`
	HomeDir     string `json:"home_dir"    gorm:"column:home_dir"`
	Description string `json:"description" gorm:"column:description"`
	Status      string `json:"status"      gorm:"column:status"`
	Raw         string `json:"raw"         gorm:"column:raw"`
}

func (a CollectAccountItem) Model(minionID int64, inet string) *model.MinionAccount {
	return &model.MinionAccount{
		MinionID:    minionID,
		Inet:        inet,
		Name:        a.Name,
		LoginName:   a.LoginName,
		UID:         a.UID,
		GID:         a.GID,
		HomeDir:     a.HomeDir,
		Description: a.Description,
		Status:      a.Status,
		Raw:         a.Raw,
	}
}

type SbomSDK struct {
	Purl      string   `json:"purl"`
	Name      string   `json:"name"`
	Version   string   `json:"version"`
	Language  string   `json:"language"`
	Algorithm string   `json:"algorithm"`
	Checksum  string   `json:"checksum"`
	Licenses  []string `json:"licenses"`
}

type SbomRequest struct {
	Filename  string      `json:"filename"    validate:"required"`
	Algorithm string      `json:"algorithm"`
	Checksum  string      `json:"checksum"    validate:"required"`
	ModifyAt  time.Time   `json:"modify_time"`
	Size      int64       `json:"size"`
	Process   SbomProcExt `json:"process"`
	SDKs      []*SbomSDK  `json:"packages"    validate:"dive"`
}

type SbomProcExt struct {
	PID      int    `json:"pid"`
	Exe      string `json:"exe"`
	Username string `json:"username"`
}

func (sr SbomRequest) Components(minionID int64, inet string, projectID int64) []*model.SBOMComponent {
	ret := make([]*model.SBOMComponent, 0, len(sr.SDKs))
	for _, sk := range sr.SDKs {
		purl := sk.Purl
		if !strings.Contains(purl, "@") {
			purl += "@0.0.0"
			sk.Version = "0.0.0"
		}
		cpt := &model.SBOMComponent{
			MinionID:  minionID,
			Inet:      inet,
			ProjectID: projectID,
			Filepath:  sr.Filename,
			SHA1:      sk.Checksum,
			Name:      sk.Name,
			Version:   sk.Version,
			Language:  sk.Language,
			Licenses:  sk.Licenses,
			PURL:      purl,
		}
		ret = append(ret, cpt)
	}

	return ret
}

type procSimple struct {
	Name       string   `json:"name,omitempty"       gorm:"column:name"`
	State      string   `json:"state,omitempty"      gorm:"column:state"`
	PID        int      `json:"pid,omitempty"        gorm:"column:pid"`
	PPID       int      `json:"ppid,omitempty"       gorm:"column:ppid"`
	PGID       int      `json:"pgid,omitempty"       gorm:"column:pgid"`
	Cmdline    string   `json:"cmdline,omitempty"    gorm:"column:cmdline"`
	Username   string   `json:"username,omitempty"   gorm:"column:username"`
	Cwd        string   `json:"cwd,omitempty"        gorm:"column:cwd"`
	Executable string   `json:"executable,omitempty" gorm:"column:executable"`
	Args       []string `json:"args,omitempty"       gorm:"column:args;json"`
}

type ProcSimples []*procSimple
