package gcode

import (
	"bytes"
	"io"
)

type Buffer struct {
	gr  Reader
	buf *bytes.Buffer
	err error
}

var _ io.Reader = &Buffer{}

func NewBuffer(r Reader) *Buffer {
	return &Buffer{gr: r, buf: new(bytes.Buffer)}
}
func (b *Buffer) Buffered() []byte { return b.buf.Bytes() }

func (b *Buffer) Read(p []byte) (int, error) {
	if b.err == io.EOF {
		return b.buf.Read(p)
	}
	if b.err != nil {
		return 0, b.err
	}

	var block Block
	for b.buf.Len() < len(p) {
		block, b.err = b.gr.Read()
		if b.err == io.EOF {
			return b.buf.Read(p)
		}
		if b.err != nil {
			return 0, b.err
		}
		b.buf.WriteString(block.String() + "\n")
	}

	return b.buf.Read(p)
}
