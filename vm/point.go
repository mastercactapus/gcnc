package vm

// import (
// 	"math"

// 	"github.com/mastercactapus/gcnc/gcode"
// )

// type Point struct{ X, Y, Z float64 }

// func (p Point) Equal(b Point) bool {
// 	return p.X == b.X && p.Y == b.Y && p.Z == b.Z
// }

// func (p Point) applyBlock(b gcode.Block, mul float64) Point {
// 	for _, g := range b {
// 		switch g.W {
// 		case 'X':
// 			p.X = g.Arg * mul
// 		case 'Y':
// 			p.Y = g.Arg * mul
// 		case 'Z':
// 			p.Z = g.Arg * mul
// 		}
// 	}

// 	return p
// }

// func (p Point) Mul(val float64) Point {
// 	p.X *= val
// 	p.Y *= val
// 	p.Z *= val
// 	return p
// }

// // Add will add the target values to p.
// func (p Point) Add(target Point) Point {
// 	p.X += target.X
// 	p.Y += target.Y
// 	p.Z += target.Z
// 	return p
// }

// // Sub will subtract the target values from p.
// func (p Point) Sub(target Point) Point {
// 	p.X -= target.X
// 	p.Y -= target.Y
// 	p.Z -= target.Z
// 	return p
// }

// // Split will return a set of evenly spaced Pointinates
// // from c to the target.
// func (p Point) Split(target Point, n int, relative bool) []Point {
// 	target.X = (target.X - p.X) / float64(n)
// 	target.Y = (target.Y - p.Y) / float64(n)
// 	target.Z = (target.Z - p.Z) / float64(n)

// 	res := make([]Point, n)
// 	for i := range res {
// 		if relative {
// 			res[i].X = target.X
// 			res[i].Y = target.Y
// 			res[i].Z = target.Z
// 		} else {
// 			res[i].X = p.X + target.X*float64(i+1)
// 			res[i].Y = p.Y + target.Y*float64(i+1)
// 			res[i].Z = p.Z + target.Z*float64(i+1)
// 		}
// 	}

// 	return res
// }

// // DistanceXY will return the X,Y distance between c and the target Pointinate.
// func (p Point) DistanceXY(target Point) float64 {
// 	return math.Sqrt(math.Pow(target.X-p.X, 2) + math.Pow(target.Y-p.Y, 2))
// }
