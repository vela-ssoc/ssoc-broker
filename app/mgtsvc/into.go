package mgtsvc

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/vela-ssoc/vela-broker/bridge/mlink"
	"github.com/vela-ssoc/vela-common-mb-itai/problem"
	"github.com/vela-ssoc/vela-common-mba/netutil"
)

type IntoService interface {
	ARR(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	AWS(ctx context.Context, w http.ResponseWriter, r *http.Request, id int64) error
}

func Into(lnk mlink.Linker) IntoService {
	name := lnk.Name()
	biz := &intoService{
		lnk:  lnk,
		name: name,
	}
	upgrade := netutil.Upgrade(biz.upgradeErrorFunc)
	biz.upgrade = upgrade

	return biz
}

type intoService struct {
	lnk     mlink.Linker
	name    string
	upgrade websocket.Upgrader
}

func (biz *intoService) ARR(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	biz.lnk.Forward(w, r)
	return nil
}

func (biz *intoService) AWS(ctx context.Context, w http.ResponseWriter, r *http.Request, id int64) error {
	path := r.URL.Path + "?" + r.URL.RawQuery
	up, _, err := biz.lnk.Stream(ctx, id, path, nil)
	if err != nil {
		return err
	}

	down, err := biz.upgrade.Upgrade(w, r, nil)
	if err != nil {
		_ = up.Close()
		return err
	}

	netutil.PipeWebsocket(up, down)

	return nil
}

func (biz *intoService) upgradeErrorFunc(w http.ResponseWriter, r *http.Request, status int, reason error) {
	pd := &problem.Detail{
		Type:     biz.name,
		Title:    "websocket 协议升级错误",
		Status:   status,
		Detail:   reason.Error(),
		Instance: r.RequestURI,
	}
	_ = pd.JSON(w)
}
