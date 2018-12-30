package vm

import (
	"errors"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
)

type Machine struct {
	pos coord.Point
	wco coord.Point

	modal [256]float64

	feed float64
}

func NewMachine() *Machine {
	m := &Machine{}

	// using grbl defaults
	m.modal[gcode.ModalGroupMotion] = 0
	m.modal[gcode.ModalGroupCoordinateSystem] = 54
	m.modal[gcode.ModalGroupPlaneSelection] = 17
	m.modal[gcode.ModalGroupDistanceMode] = 90
	m.modal[gcode.ModalGroupArcDistanceMode] = 91.1
	m.modal[gcode.ModalGroupFeedRateMode] = 94
	m.modal[gcode.ModalGroupUnits] = 21
	m.modal[gcode.ModalGroupCutterCompensationMode] = 40
	m.modal[gcode.ModalGroupToolLength] = 49
	m.modal[gcode.ModalGroupStopping] = 0
	m.modal[gcode.ModalGroupSpindle] = 5
	m.modal[gcode.ModalGroupCoolant] = 9

	return m
}

func (m Machine) Inches() bool         { return m.modal[gcode.ModalGroupUnits] == 20 }
func (m Machine) RelativeMotion() bool { return m.modal[gcode.ModalGroupDistanceMode] == 91 }

func (m Machine) WPos() coord.Point {
	return m.pos.Sub(m.wco)
}
func (m Machine) MPos() coord.Point {
	return m.pos
}
func (m Machine) WCO() coord.Point {
	return m.wco
}

func isSupported(g gcode.Word) bool {
	if g.IsAxis() {
		return true
	}

	if g.W == 'G' {
		switch g.Arg {
		case 0, 1, 91, 90, 20, 21, 94:
			return true
		}
	} else if g.W == 'F' {
		return true
	} else if g.W == 'M' {
		switch g.Arg {
		case 3, 5:
			return true
		}
	}

	return false
}

func applyBlock(p coord.Point, b gcode.Block, mul float64) coord.Point {
	for _, g := range b {
		switch g.W {
		case 'X':
			p.X = g.Arg * mul
		case 'Y':
			p.Y = g.Arg * mul
		case 'Z':
			p.Z = g.Arg * mul
		}
	}

	return p
}

func (m *Machine) Run(b gcode.Block) error {
	err := b.Validate()
	if err != nil {
		return err
	}
	var machineCoords bool
	for _, g := range b {
		mg := g.ModalGroup()
		if mg != gcode.ModalGroupNone && mg != gcode.ModalGroupNonModal {
			m.modal[mg] = g.Arg
		}
		if g == (gcode.Word{W: 'G', Arg: 53.0}) {
			machineCoords = true
		}
		if !isSupported(g) {
			return errors.New("unsupported code: " + g.String())
		}
	}

	args := b.Args()
	if len(args) == 0 {
		return nil
	}

	mul := 1.0
	if m.Inches() {
		mul = 2.54
	}
	// apply motion
	if m.RelativeMotion() {
		m.pos = m.pos.Add(applyBlock(coord.Point{}, args, mul))
	} else if machineCoords {
		m.pos = applyBlock(m.pos, args, 1)
	} else {
		m.pos = applyBlock(m.WPos(), args, mul).Add(m.wco)
	}

	return nil
}
