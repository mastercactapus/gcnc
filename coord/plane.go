package coord

type Plane [3]Point

func (p Plane) Z(x, y float64) float64 {
	a := p[0].Y*(p[1].Z-p[2].Z) + p[1].Y*(p[2].Z-p[0].Z) + p[2].Y*(p[0].Z-p[1].Z)
	b := p[0].Z*(p[1].X-p[2].X) + p[1].Z*(p[2].X-p[0].X) + p[2].Z*(p[0].X-p[1].X)
	c := p[0].X*(p[1].Y-p[2].Y) + p[1].X*(p[2].Y-p[0].Y) + p[2].X*(p[0].Y-p[1].Y)
	d := -p[0].X*(p[1].Y*p[2].Z-p[2].Y*p[1].Z) - p[1].X*(p[2].Y*p[0].Z-p[0].Y*p[2].Z) - p[2].X*(p[0].Y*p[1].Z-p[1].Y*p[0].Z)

	return -(d - a*x - b*y) / c
}
