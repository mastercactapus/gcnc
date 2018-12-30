package meshlevel

import (
	"errors"
	"math"

	"github.com/fogleman/delaunay"
	"github.com/mastercactapus/gcnc/coord"
)

type Mesh struct {
	minX, minY, maxX, maxY float64
	triangles              []coord.Triangle
}

func NewMesh(points []coord.Point) (*Mesh, error) {
	if len(points) < 3 {
		return nil, errors.New("need at least 3 points to create a mesh")
	}

	points2d := make([]delaunay.Point, len(points))
	m := make(map[delaunay.Point]coord.Point, len(points))

	mesh := &Mesh{
		minX: points[0].X,
		minY: points[0].Y,
		maxX: points[0].X,
		maxY: points[0].Y,
	}
	var d delaunay.Point
	for i, p := range points {
		mesh.minX = math.Min(mesh.minX, p.X)
		mesh.minY = math.Min(mesh.minY, p.Y)
		mesh.maxX = math.Max(mesh.maxX, p.X)
		mesh.maxY = math.Max(mesh.maxY, p.Y)

		d.X = p.X
		d.Y = p.Y
		m[d] = p
		points2d[i] = d
	}
	mesh.minX -= coord.Epsilon
	mesh.minY -= coord.Epsilon
	mesh.maxX += coord.Epsilon
	mesh.maxY += coord.Epsilon

	tri, err := delaunay.Triangulate(points2d)
	if err != nil {
		return nil, err
	}

	mesh.triangles = make([]coord.Triangle, 0, len(tri.Triangles)/3)

	for i := 0; i < len(tri.Triangles); i += 3 {
		mesh.triangles = append(mesh.triangles, coord.Triangle{
			A: m[tri.Points[tri.Triangles[i]]],
			B: m[tri.Points[tri.Triangles[i+1]]],
			C: m[tri.Points[tri.Triangles[i+2]]],
		})
	}

	return mesh, nil
}

func (m Mesh) OffsetZ(x, y float64) (bool, float64) {
	if x < m.minX || m.maxX < x || y < m.minY || m.maxY < y {
		return false, 0
	}
	for _, t := range m.triangles {
		if !t.ContainsXY(x, y) {
			continue
		}
		return true, t.Z(x, y)
	}

	return false, 0
}
