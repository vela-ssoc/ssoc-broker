package mlink

import (
	"time"

	"github.com/vela-ssoc/vela-broker/bridge/gateway"
	"github.com/vela-ssoc/vela-common-mb/taskpool"
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

func newAsyncPhase(ph NodePhaser, pool taskpool.Executor) NodePhaser {
	return &phaseProxy{
		phase: ph,
		pool:  pool,
	}
}

type phaseProxy struct {
	phase NodePhaser
	pool  taskpool.Executor
}

func (pp *phaseProxy) Created(id int64, inet string, at time.Time) {
	if pp.phase == nil {
		return
	}
	fn := func() {
		pp.phase.Created(id, inet, at)
	}
	pp.pool.Submit(taskpool.RunnerFunc(fn))
}

func (pp *phaseProxy) Repeated(id int64, ident gateway.Ident, at time.Time) {
	if pp.phase == nil {
		return
	}
	fn := func() {
		pp.phase.Repeated(id, ident, at)
	}
	pp.pool.Submit(taskpool.RunnerFunc(fn))
}

func (pp *phaseProxy) Connected(lnk Linker, ident gateway.Ident, issue gateway.Issue, at time.Time) {
	if pp.phase == nil {
		return
	}
	fn := func() {
		pp.phase.Connected(lnk, ident, issue, at)
	}
	pp.pool.Submit(taskpool.RunnerFunc(fn))
}

func (pp *phaseProxy) Disconnected(lnk Linker, ident gateway.Ident, issue gateway.Issue, at time.Time, du time.Duration) {
	if pp.phase == nil {
		return
	}
	fn := func() {
		pp.phase.Disconnected(lnk, ident, issue, at, du)
	}
	pp.pool.Submit(taskpool.RunnerFunc(fn))
}
