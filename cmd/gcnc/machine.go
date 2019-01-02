package main

import (
	"github.com/mastercactapus/gcnc/coord"
)

type Machine interface {
	Run([]string) error
	RunLevel([]string, float64, []coord.Point) error

	ProbeGrid(opt GridOptions) ([]ProbeResult, error)
	ProbeZ(ProbeOptions) (*ProbeResult, error)
	ToolChange(ToolChangeOptions) error

	State() chan MachineState
	HoldMessage() chan string
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
