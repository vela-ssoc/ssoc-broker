package mlink

import (
	"time"

	"github.com/vela-ssoc/vela-broker/bridge/gateway"
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
