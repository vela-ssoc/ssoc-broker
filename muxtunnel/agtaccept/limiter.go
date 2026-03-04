package agtaccept

// Limiter 上线限流器，防止 N 多个节点同时蜂拥上线，
// 对 broker 节点造成压力。
type Limiter interface {
	Allowed() bool
}
