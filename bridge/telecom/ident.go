package telecom

import (
	"encoding/json"
	"net"
	"time"

	"github.com/vela-ssoc/vela-common-mb/dal/model"
	"github.com/vela-ssoc/vela-common-mba/encipher"
)

type Hide struct {
	ID      int64        `json:"id"`
	Secret  string       `json:"secret"`
	Semver  model.Semver `json:"semver"`
	Servers Addresses    `json:"servers"`
}

type Ident struct {
	ID         int64        `json:"id"`         // ID
	Secret     string       `json:"secret"`     // 密钥
	Semver     model.Semver `json:"semver"`     // 版本
	Inet       net.IP       `json:"inet"`       // IPv4 地址
	MAC        string       `json:"mac"`        // MAC 地址
	Goos       string       `json:"goos"`       // runtime.GOOS
	Arch       string       `json:"arch"`       // runtime.GOARCH
	CPU        int          `json:"cpu"`        // runtime.NumCPU
	PID        int          `json:"pid"`        // os.Getpid
	Workdir    string       `json:"workdir"`    // os.Getwd
	Executable string       `json:"executable"` // os.Executable
	Username   string       `json:"username"`   // user.Current
	Hostname   string       `json:"hostname"`   // os.Hostname
	TimeAt     time.Time    `json:"time_at"`    // 发起时间
}

func (ide Ident) encrypt() ([]byte, error) {
	return encipher.EncryptJSON(ide)
}

func (ide Ident) String() string {
	dat, _ := json.MarshalIndent(ide, "", "    ")
	return string(dat)
}
