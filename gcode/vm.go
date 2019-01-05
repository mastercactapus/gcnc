package gcode

import (
	"errors"

	"github.com/mastercactapus/gcnc/coord"
)

// VM will track state and interpret gcode.
type VM struct {
	pos coord.Point
	wco coord.Point

	modal [256]float64

	feed float64
}

// NewVM constructs a new VM with default state.
func NewVM() *VM {
	vm := &VM{}

	// using grbl defaults
	vm.modal[ModalGroupMotion] = 0
	vm.modal[ModalGroupCoordinateSystem] = 54
	vm.modal[ModalGroupPlaneSelection] = 17
	vm.modal[ModalGroupDistanceMode] = 90
	vm.modal[ModalGroupArcDistanceMode] = 91.1
	vm.modal[ModalGroupFeedRateMode] = 94
	vm.modal[ModalGroupUnits] = 21
	vm.modal[ModalGroupCutterCompensationMode] = 40
	vm.modal[ModalGroupToolLength] = 49
	vm.modal[ModalGroupStopping] = 0
	vm.modal[ModalGroupSpindle] = 5
	vm.modal[ModalGroupCoolant] = 9

	return vm
}

func (vm VM) Inches() bool         { return vm.modal[ModalGroupUnits] == 20 }
func (vm VM) RelativeMotion() bool { return vm.modal[ModalGroupDistanceMode] == 91 }

func (vm VM) WPos() coord.Point {
	return vm.pos.Sub(vm.wco)
}
func (vm VM) MPos() coord.Point {
	return vm.pos
}
func (vm *VM) SetMPos(p coord.Point) {
	vm.pos = p
}
func (vm *VM) SetWCO(p coord.Point) {
	vm.wco = p
}
func (vm VM) WCO() coord.Point {
	return vm.wco
}

func isSupported(g Word) bool {
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

func applyBlock(p coord.Point, b Block, mul float64) coord.Point {
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

func (vm *VM) Run(b Block) error {
	err := b.Validate()
	if err != nil {
		return err
	}
	var machineCoords bool
	for _, g := range b {
		mg := g.ModalGroup()
		if mg != ModalGroupNone && mg != ModalGroupNonModal {
			vm.modal[mg] = g.Arg
		}
		if g == (Word{W: 'G', Arg: 53.0}) {
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
	if vm.Inches() {
		mul = 2.54
	}
	// apply motion
	if vm.RelativeMotion() {
		vm.pos = vm.pos.Add(applyBlock(coord.Point{}, args, mul))
	} else if machineCoords {
		vm.pos = applyBlock(vm.pos, args, 1)
	} else {
		vm.pos = applyBlock(vm.WPos(), args, mul).Add(vm.wco)
	}

	return nil
}
