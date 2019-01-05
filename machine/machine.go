package machine

import (
	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
)

type Machine struct {
	Adapter

	holdMessage chan string
}
type State struct {
	Status string
	MPos   coord.Point
	WCO    coord.Point
}

func NewMachine(a Adapter) *Machine {
	return &Machine{
		Adapter:     a,
		holdMessage: make(chan string),
	}
}

func (m *Machine) HoldMessage() chan string {
	return m.holdMessage
}

func (m *Machine) runBlocks(b []gcode.Block) error {
	_, err := m.Adapter.ReadFrom(gcode.NewBuffer(&gcode.BlocksReader{Blocks: b}))
	return err
}

func (m *Machine) hold(message string) error {
	m.holdMessage <- message
	_, err := m.Adapter.Write([]byte("M0\n"))
	m.holdMessage <- "-"
	return err
}

func generateGoTo(travelZ float64, pos coord.Point) []gcode.Block {
	return []gcode.Block{
		{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'Z', Arg: travelZ},
		},
		{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'X', Arg: pos.X},
			{W: 'Y', Arg: pos.Y},
		},
		{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'Z', Arg: pos.Z},
		},
	}
}
