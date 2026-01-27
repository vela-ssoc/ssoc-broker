package config

type Hide struct {
	Secret    string   `json:"secret"    validate:"required,lte=1000"`
	Protocol  string   `json:"protocol"  validate:"omitempty,oneof=smux yamux"`
	Addresses []string `json:"addresses" validate:"gte=1,lte=100,dive,required"`
	Semver    string   `json:"semver"    validate:"semver"`
}
