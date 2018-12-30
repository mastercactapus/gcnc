package coord

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTriangle_Z(t *testing.T) {
	tri := Triangle{
		A: Point{0, 0, 0},
		B: Point{10, 0, 0},
		C: Point{5, 5, 5},
	}

	res := tri.Z(0, 0)
	assert.Equal(t, 0.0, res)

	res = tri.Z(5, 0)
	assert.Equal(t, 0.0, res)

	res = tri.Z(5, 5)
	assert.Equal(t, 5.0, res)

	res = tri.Z(2.5, 2.5)
	assert.Equal(t, 2.5, res)
}
