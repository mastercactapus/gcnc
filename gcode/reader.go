package gcode

import "io"

type Reader interface {
	Read() (Block, error)
}

type BlocksReader struct {
	Blocks []Block
	n      int
}

func (b *BlocksReader) Read() (Block, error) {
	if b.n == len(b.Blocks) {
		return nil, io.EOF
	}

	b.n++
	return b.Blocks[b.n-1], nil
}
