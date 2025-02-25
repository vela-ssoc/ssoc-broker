package mgtapi

import (
	"os"

	"github.com/xgfone/ship/v5"
)

func NewSystem() *System {
	return &System{}
}

type System struct{}

func (sys *System) exit(_ *ship.Context) error {
	os.Exit(0)
	return nil
}
