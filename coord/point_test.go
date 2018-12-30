package coord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPoint_Add(t *testing.T) {
	a := Point{X: 1, Y: 2, Z: 3}
	b := Point{X: 4, Y: 5, Z: 6}

	assert.Equal(t, Point{X: 5, Y: 7, Z: 9}, a.Add(b))
}

func TestPoint_DistanceXY(t *testing.T) {
	dist := Point{X: 1, Y: 2, Z: 3}.DistanceXY(4,5)
	assert.InEpsilon(t, 4.24264, dist, .01)
}

func TestPoint_Split(t *testing.T) {
	var a Point //zero
	b := Point{X: 10, Y: 10, Z: 10}

	res := a.Split(b, 2, false)

	assert.Equal(t, []Point{{X: 5, Y: 5, Z: 5}, {X: 10, Y: 10, Z: 10}}, res)

	a = Point{X: 10, Y: 10, Z: 10}
	b = Point{X: 20, Y: 20, Z: 20}
	res = a.Split(b, 4, false)
	assert.Equal(t,
		[]Point{{X: 12.5, Y: 12.5, Z: 12.5}, {X: 15, Y: 15, Z: 15}, {X: 17.5, Y: 17.5, Z: 17.5}, {X: 20, Y: 20, Z: 20}},
		res,
	)

}
