package param

type UpgradeDownload struct {
	Tags       []string `json:"tags"       query:"tags"`
	Version    string   `json:"version"    query:"version" validate:"omitempty,semver"`
	Customized string   `json:"customized" query:"customized"`
}
