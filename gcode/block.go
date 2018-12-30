package gcode

import (
	"errors"
)

type Block []Word

func (b Block) Arg(w byte) (bool, float64) {
	for _, g := range b {
		if g.W == w {
			return true, g.Arg
		}
	}
	return false, 0
}
func (b Block) SetArg(w byte, val float64) {
	for i, g := range b {
		if g.W == w {
			b[i].Arg = val
			return
		}
	}
}

func (b Block) Args() Block {
	res := make(Block, 0, len(b))
	for _, g := range b {
		if g.ModalGroup() == ModalGroupNone {
			res = append(res, g)
		}
	}
	return res
}
func (b Block) Clone() Block {
	c := make(Block, len(b))
	copy(c, b)
	return c
}

func (b Block) HasModal() bool {
	for _, g := range b {
		if g.ModalGroup() != ModalGroupNone {
			return true
		}
	}
	return false
}

func (b Block) Validate() error {
	var checkWord [256]bool
	var checkModal [256]bool

	var m ModalGroup
	for _, g := range b {
		if !g.IsValid() {
			return errors.New("invalid word in block")
		}
		if g.W != 'G' && checkWord[g.W] {
			return errors.New("word was repeated in a block")
		}
		checkWord[g.W] = true
		m = g.ModalGroup()
		if m != ModalGroupNone && checkModal[m] {
			return errors.New("multiple words from same modal group")
		}
		checkModal[m] = true
	}

	return nil
}
