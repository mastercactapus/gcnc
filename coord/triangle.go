package coord

import "math"

const (
	// Epsilon is the max error when checking containment.
	Epsilon   = 0.001
	epsilonSq = Epsilon * Epsilon
)

type Triangle struct{ A, B, C Point }

// ContainsXY returns true if the 2D projection of the triangle
// has the point x,y.
func (t Triangle) ContainsXY(x, y float64) bool {
	return accuratePointInTriangle(
		t.A.X, t.A.Y,
		t.B.X, t.B.Y,
		t.C.X, t.C.Y,
		x, y)
}

// Z will give the Z-coordinate on the plane defined by the triangle
// where it intersects x,y.
func (t Triangle) Z(x, y float64) float64 {
	a := t.A.Y*(t.B.Z-t.C.Z) + t.B.Y*(t.C.Z-t.A.Z) + t.C.Y*(t.A.Z-t.B.Z)
	b := t.A.Z*(t.B.X-t.C.X) + t.B.Z*(t.C.X-t.A.X) + t.C.Z*(t.A.X-t.B.X)
	c := t.A.X*(t.B.Y-t.C.Y) + t.B.X*(t.C.Y-t.A.Y) + t.C.X*(t.A.Y-t.B.Y)
	d := -t.A.X*(t.B.Y*t.C.Z-t.C.Y*t.B.Z) - t.B.X*(t.C.Y*t.A.Z-t.A.Y*t.C.Z) - t.C.X*(t.A.Y*t.B.Z-t.B.Y*t.A.Z)

	return (d - a*x - b*y) / c
}

// adapted from https://totologic.blogspot.com/2014/01/accurate-point-in-triangle-test.html

func side(x1, y1, x2, y2, x, y float64) float64 {
	return (y2-y1)*(x-x1) + (-x2+x1)*(y-y1)
}
func naivePointInTriangle(x1, y1, x2, y2, x3, y3, x, y float64) bool {
	checkSide1 := side(x1, y1, x2, y2, x, y) >= 0
	checkSide2 := side(x2, y2, x3, y3, x, y) >= 0
	checkSide3 := side(x3, y3, x1, y1, x, y) >= 0
	return checkSide1 && checkSide2 && checkSide3
}
func pointInTriangleBoundingBox(x1, y1, x2, y2, x3, y3, x, y float64) bool {
	xMin := math.Min(x1, math.Min(x2, x3)) - Epsilon
	xMax := math.Max(x1, math.Max(x2, x3)) + Epsilon
	yMin := math.Min(y1, math.Min(y2, y3)) - Epsilon
	yMax := math.Max(y1, math.Max(y2, y3)) + Epsilon

	if x < xMin || xMax < x || y < yMin || yMax < y {
		return false
	}
	return true
}

func distanceSquarePointToSegment(x1, y1, x2, y2, x, y float64) float64 {
	p1p2squareLength := (x2-x1)*(x2-x1) + (y2-y1)*(y2-y1)
	dotProduct := ((x-x1)*(x2-x1) + (y-y1)*(y2-y1)) / p1p2squareLength
	if dotProduct < 0 {
		return (x-x1)*(x-x1) + (y-y1)*(y-y1)
	}
	if dotProduct <= 1 {
		p0p1squareLength := (x1-x)*(x1-x) + (y1-y)*(y1-y)
		return p0p1squareLength - dotProduct*dotProduct*p1p2squareLength
	}

	return (x-x2)*(x-x2) + (y-y2)*(y-y2)
}

func accuratePointInTriangle(x1, y1, x2, y2, x3, y3, x, y float64) bool {
	if !pointInTriangleBoundingBox(x1, y1, x2, y2, x3, y3, x, y) {
		return false
	}

	if naivePointInTriangle(x1, y1, x2, y2, x3, y3, x, y) {
		return true
	}
	if distanceSquarePointToSegment(x1, y1, x2, y2, x, y) <= epsilonSq {
		return true
	}
	if distanceSquarePointToSegment(x2, y2, x3, y3, x, y) <= epsilonSq {
		return true
	}
	if distanceSquarePointToSegment(x3, y3, x1, y1, x, y) <= epsilonSq {
		return true
	}

	return false
}
