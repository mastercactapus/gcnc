package gcode

import (
	"bytes"
	"io"
)

func Parse(data string) ([]Block, error) {
	r := NewParser(bytes.NewBufferString(data))
	var b []Block
	for {
		bl, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		b = append(b, bl)
	}
	return b, nil
}

func MustParse(data string) []Block {
	b, err := Parse(data)
	if err != nil {
		panic(err)
	}
	return b
}
