package mservice

import "os"

func NewSystem() *System {
	return &System{}
}

type System struct{}

func (sys *System) Exit() {
	os.Exit(0)
}
