package gcode

import (
	"strconv"
	"strings"
)

type Word struct {
	W   byte
	Arg float64
}

func (w Word) IsAxis() bool {
	switch w.W {
	case 'X', 'Y', 'Z': // maybe someday 'A', 'B', 'C', 'U', 'V', 'W':
		return true
	}
	return false
}

func (w Word) IsValid() bool {
	return w.W >= 'A' && w.W <= 'Z'
}

func formatFloat(f float64, prec int) string {
	s := strconv.FormatFloat(f, 'f', prec, 64)
	if strings.ContainsRune(s, '.') {
		s = strings.TrimRight(s, "0")
	}
	return strings.TrimRight(s, ".")
}

func (w Word) String() string {
	return string(w.W) + formatFloat(w.Arg, 3)
}
