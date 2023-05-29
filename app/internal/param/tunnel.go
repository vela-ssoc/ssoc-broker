package param

type TunnelTCP struct {
	Address string `json:"address" query:"address" validate:"hostname_port"`
}
