package meshlevel

import (
	"github.com/mastercactapus/gcnc/coord"
)

func OffsetFrom(z float64, points []coord.Point) []coord.Point {
	p := make([]coord.Point, len(points))
	copy(p, points)

	for i := range p {
		p[i].Z -= z
	}
	return p
}
