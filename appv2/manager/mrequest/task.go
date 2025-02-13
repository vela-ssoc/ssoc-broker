package mrequest

import "time"

type TaskPushData struct {
	// ID 即任务的唯一标识。
	ID int64 `json:"id"`

	// ExecID 任务的执行 ID。
	// 同一个任务可以被多次触发执行，每次执行时都会生成一个新的 ExecID，
	// 用于标识任务执行的不同批次。
	ExecID int64 `json:"exec_id"`

	// Name 任务名，比如：内网 Log4J 扫描。
	// 注意：当前中心端要求任务 Name 唯一，但是 agent 尽量不要直接拿 Name 区分唯一性，
	// 一是业务可能会变化，二是任务可以删除新建，名字可能相同。
	// 唯一性判断请以 ID 为准。
	Name string `json:"name"`

	// Intro 任务简介给人看的，对程序处理来说无实际意义。
	Intro string `json:"intro"`

	// Code 可运行的 Lua 代码。
	Code string `json:"code"`

	// CodeSHA1 Lua 代码的 SHA-1 值（小写）。
	CodeSHA1 string `json:"code_sha1"`

	// Timeout 任务超时时间。
	// 注意：该时间可能为 0 （即：未指定超时时间），对此 agent 可自行
	Timeout time.Duration `json:"timeout"`
}

type TaskPush struct {
	ExecID int64 `json:"exec_id"`
}
