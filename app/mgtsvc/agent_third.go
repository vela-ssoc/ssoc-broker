package mgtsvc

import (
	"context"

	"github.com/vela-ssoc/ssoc-common-mb/accord"
)

func (biz *agentService) ThirdDiff(_ context.Context, name, event string) error {
	req := &accord.ThirdDiff{Name: name, Event: event}
	go biz.broadcast("/api/v1/agent/third/diff", req)
	return nil
}
