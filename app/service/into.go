package service

import (
	"context"
	"net/http"

	"github.com/vela-ssoc/vela-broker/bridge/mlink"
)

type IntoService interface {
	ARR(ctx context.Context, w http.ResponseWriter, r *http.Request) error
	AWS(ctx context.Context, w http.ResponseWriter, r *http.Request) error
}

func Into(lnk mlink.Linker) IntoService {
	return &intoService{
		lnk: lnk,
	}
}

type intoService struct {
	lnk mlink.Linker
}

func (biz *intoService) ARR(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	biz.lnk.Forward(w, r)
	return nil
}

func (biz *intoService) AWS(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return nil
}
