package serverd

type authRequest struct {
	MachineID  string `json:"machine_id" validate:"required"`    // 机器码
	Inet       string `json:"inet"       validate:"required,ip"` // 出口 IP
	PID        int    `json:"pid"`                               // 进程 PID
	Workdir    string `json:"workdir"`                           // 工作目录
	Executable string `json:"executable"`                        // 执行路径
	Hostname   string `json:"hostname"`                          // 主机名
	Goos       string `json:"goos"`                              // runtime.GOOS
	Goarch     string `json:"goarch"`                            // runtime.GOARCH
	Semver     string `json:"semver"`                            // 节点版本
	Unload     bool   `json:"unload"`                            // 是否开启静默模式，仅在新注册节点时有效
	Unstable   bool   `json:"unstable"`                          // 不稳定版本
	Customized string `json:"customized"`                        // 定制版本
}

type authResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
