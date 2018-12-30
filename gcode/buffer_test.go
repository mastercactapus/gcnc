package gcode

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffer_Read(t *testing.T) {
	blocks := []Block{
		{{W: 'G', Arg: 1}, {W: 'G', Arg: 2}},

		{{W: 'M', Arg: 2}},
	}

	gr := &BlocksReader{Blocks: blocks}

	b := NewBuffer(gr)

	buf := make([]byte, 10)
	n, err := b.Read(buf)
	assert.NoError(t, err)
	assert.Equal(t, 8, n)
	assert.Equal(t, []byte("G1G2\nM2\n"), buf[:n])

	n, err = b.Read(buf)
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)
}
