package mlink

import (
	"time"

	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-common-mb/gopool"
)

type NodePhaser interface {
	// Created 新节点注册
	Created(id int64, inet string, at time.Time)

	// Repeated 节点重复登录
	Repeated(id int64, ident gateway.Ident, at time.Time)

	// Connected 节点连接成功
	Connected(lnk Linker, ident gateway.Ident, issue gateway.Issue, at time.Time)

	// Disconnected 节点断开连接
	Disconnected(lnk Linker, ident gateway.Ident, issue gateway.Issue, at time.Time, du time.Duration)
}

func newAsyncPhase(ph NodePhaser, pool gopool.Executor) NodePhaser {
	return &phaseProxy{
		phase: ph,
		pool:  pool,
	}
}

type phaseProxy struct {
	phase NodePhaser
	pool  gopool.Executor
}

func (pp *phaseProxy) Created(id int64, inet string, at time.Time) {
	if pp.phase == nil {
		return
	}
	fn := func() {
		pp.phase.Created(id, inet, at)
	}
	pp.pool.Execute(fn)
}

func (pp *phaseProxy) Repeated(id int64, ident gateway.Ident, at time.Time) {
	if pp.phase == nil {
		return
	}
	fn := func() {
		pp.phase.Repeated(id, ident, at)
	}
	pp.pool.Execute(fn)
}

func (pp *phaseProxy) Connected(lnk Linker, ident gateway.Ident, issue gateway.Issue, at time.Time) {
	if pp.phase == nil {
		return
	}
	fn := func() {
		pp.phase.Connected(lnk, ident, issue, at)
	}
	pp.pool.Execute(fn)
}

func (pp *phaseProxy) Disconnected(lnk Linker, ident gateway.Ident, issue gateway.Issue, at time.Time, du time.Duration) {
	if pp.phase == nil {
		return
	}
	fn := func() {
		pp.phase.Disconnected(lnk, ident, issue, at, du)
	}
	pp.pool.Execute(fn)
}
