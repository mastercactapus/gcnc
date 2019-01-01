package main

import (
	"github.com/mastercactapus/gcnc/coord"
)

type Machine interface {
	Run([]string) error
	ProbeGrid(opt ProbeOptions, xDist, yDist float64) ([]ProbeResult, error)
	ProbeZ(ProbeOptions) (*ProbeResult, error)

	State() chan MachineState
}
type ProbeOptions struct {
	ZeroZAxis bool
	FeedRate  float64
	MaxTravel float64
}
type ProbeResult struct {
	coord.Point
	Valid bool
}
type MachineState struct {
	Status string
	MPos   coord.Point
	WCO    coord.Point
}
