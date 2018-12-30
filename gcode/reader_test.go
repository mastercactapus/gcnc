package gcode

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBlocksReader(t *testing.T) {
	blocks := []Block{
		{{W: 'G', Arg: 1}, {W: 'G', Arg: 2}},

		{{W: 'M', Arg: 2}},
	}

	gr := &BlocksReader{Blocks: blocks}

	b, err := gr.Read()
	assert.NoError(t, err)
	assert.Equal(t, Block{{W: 'G', Arg: 1}, {W: 'G', Arg: 2}}, b)

	b, err = gr.Read()
	assert.NoError(t, err)
	assert.Equal(t, Block{{W: 'M', Arg: 2}}, b)

	b, err = gr.Read()
	assert.Error(t, err)
	assert.Equal(t, io.EOF, err)
	assert.Nil(t, b)
}
