package response

type TunnelStat struct {
	RX        uint64  `json:"rx"`
	TX        uint64  `json:"tx"`
	Limit     float64 `json:"limit"`
	Unlimited bool    `json:"unlimited"`
}
