package meshlevel

import (
	"math"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
	"github.com/mastercactapus/gcnc/vm"
)

type MeshLeveler struct {
	minX, maxX, minY, maxY float64

	offsets     []coord.Point
	granularity float64

	buf  []gcode.Block
	bufN int

	splitVM *vm.Machine
	levelVM *vm.Machine

	gr gcode.Reader
}

func New(offsets []coord.Point, granularity float64, gr gcode.Reader) *MeshLeveler {
	l := &MeshLeveler{
		offsets: offsets,

		minX: offsets[0].X,
		maxX: offsets[0].X,
		minY: offsets[0].Y,
		maxY: offsets[0].Y,

		splitVM: vm.NewMachine(),
		levelVM: vm.NewMachine(),

		granularity: granularity,
		gr:          gr,
	}

	for _, p := range offsets[1:] {
		if p.X < l.minX {
			l.minX = p.X
		}
		if p.X > l.maxX {
			l.maxX = p.X
		}
		if p.Y < l.minY {
			l.minY = p.Y
		}
		if p.Y > l.maxY {
			l.maxY = p.Y
		}
	}

	return l
}

func (l MeshLeveler) Offset(x, y float64) float64 {
	if x < l.minX || x > l.maxX {
		return 0
	}
	if y < l.minY || y > l.maxY {
		return 0
	}

	a, b, c := l.offsets[0], l.offsets[1], l.offsets[2]
	dA, dB, dC := a.DistanceXY(x, y), b.DistanceXY(x, y), c.DistanceXY(x, y)

	for _, p := range l.offsets {
		d := p.DistanceXY(x, y)
		if d < dA {
			a = p
			dA = d
			continue
		}
		if d < dB {
			b = p
			dB = d
			continue
		}
		if d < dC {
			b = p
			dC = d
			continue
		}
	}

	pl := coord.Plane{a, b, c}
	return pl.Z(x, y)
}

func (l *MeshLeveler) Read() (gcode.Block, error) {
	b, err := l.next()
	if err != nil {
		return nil, err
	}

	oldPos := l.levelVM.MPos()
	err = l.levelVM.Run(b)
	if err != nil {
		return nil, err
	}
	newPos := l.levelVM.MPos()
	if oldPos.Equal(newPos) {
		return b, nil
	}

	oldOffset := l.Offset(oldPos.X, oldPos.Y)
	newOffset := l.Offset(newPos.X, newPos.Y)
	if oldOffset == newOffset {
		return b, nil
	}

	b = b.Clone()
	ok, oldZ := b.Arg('Z')
	if !l.levelVM.RelativeMotion() && !ok {
		oldZ = oldPos.Z
	}

	if !ok {
		b = append(b, gcode.Word{W: 'Z', Arg: newOffset - oldOffset})
	} else {
		b.SetArg('Z', oldZ+(newOffset-oldOffset))
	}

	return b, nil
}

func (l *MeshLeveler) next() (gcode.Block, error) {
	if len(l.buf)-l.bufN > 0 {
		l.bufN++
		return l.buf[l.bufN-1], nil
	}
	b, err := l.gr.Read()
	if err != nil {
		return nil, err
	}

	oldPos := l.splitVM.MPos()
	err = l.splitVM.Run(b)
	if err != nil {
		return nil, err
	}
	newPos := l.splitVM.MPos()
	if oldPos.Equal(newPos) {
		return b, nil
	}
	dist := oldPos.DistanceXY(newPos.X, newPos.Y)
	if dist <= l.granularity {
		return b, nil
	}

	n := int(math.Ceil(dist / l.granularity))
	distPoint := newPos.Sub(oldPos).Div(float64(n))

	if l.splitVM.RelativeMotion() {
		bl := b.Clone()
		bl.SetArg('X', distPoint.X)
		bl.SetArg('Y', distPoint.Y)
		bl.SetArg('Z', distPoint.Z)

		for i := 0; i < n; i++ {
			l.buf = append(l.buf, bl)
		}
	} else {
		for i := 0; i < n; i++ {
			bl := b.Clone()
			bl.SetArg('X', oldPos.X+distPoint.X*float64(i))
			bl.SetArg('Y', oldPos.Y+distPoint.Y*float64(i))
			bl.SetArg('Z', oldPos.Z+distPoint.Z*float64(i))

			l.buf = append(l.buf, bl)
		}
	}

	l.bufN = 1
	return l.buf[0], nil
}
