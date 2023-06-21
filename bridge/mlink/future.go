package mlink

import (
	"context"
	"sync"
	"time"
)

type Future struct {
	mid int64
	err error
}

func (f *Future) MinionID() int64 { return f.mid }
func (f *Future) Error() error    { return f.err }

type futureTask struct {
	wg   *sync.WaitGroup
	hub  *minionHub
	mid  int64
	path string
	req  any
	ret  chan<- *Future
}

func (ft *futureTask) Run() {
	defer ft.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ft.hub.silentJSON(ctx, ft.mid, ft.path, ft.req)
	fut := &Future{mid: ft.mid, err: err}
	ft.ret <- fut
}

type onewayTask struct {
	wg   *sync.WaitGroup
	hub  *minionHub
	mid  int64
	path string
	req  any
	err  error
}

func (ot *onewayTask) Run() {
	defer ot.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ot.err = ot.hub.silentJSON(ctx, ot.mid, ot.path, ot.req)
}

func (ot *onewayTask) Wait() error {
	ot.wg.Wait()
	return ot.err
}

type unicastTask struct {
	wg   *sync.WaitGroup
	hub  *minionHub
	mid  int64
	path string
	req  any
	resp any
	err  error
}

func (ut *unicastTask) Run() {
	defer ut.wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := ut.hub.json(ctx, ut.mid, ut.path, ut.req, ut.resp)
	ut.err = err
}

func (ut *unicastTask) Wait() error {
	ut.wg.Wait()
	return ut.err
}
