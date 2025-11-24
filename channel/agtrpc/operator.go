package agtrpc

import "context"

type Operator interface {
	// Command 向 agent 发送简单指令。
	Command(ctx context.Context, cmd string) error
}
