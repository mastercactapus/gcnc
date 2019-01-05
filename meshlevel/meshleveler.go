package meshlevel

import (
	"math"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
)

type MeshLeveler struct {
	granularity float64
	offsetter   ZOffsetter

	buf  []gcode.Block
	bufN int

	splitVM *gcode.VM
	levelVM *gcode.VM

	gr gcode.Reader
}
type Config struct {
	ZOffsetter  ZOffsetter
	Granularity float64

	MPos, WCO coord.Point

	Reader gcode.Reader
}

func New(cfg Config) *MeshLeveler {
	l := &MeshLeveler{

		splitVM: gcode.NewVM(),
		levelVM: gcode.NewVM(),

		granularity: cfg.Granularity,
		gr:          cfg.Reader,

		offsetter: cfg.ZOffsetter,
	}
	if l.offsetter == nil {
		l.offsetter = dummyOffsetter{}
	}
	l.splitVM.SetMPos(cfg.MPos)
	l.levelVM.SetMPos(cfg.MPos)

	l.splitVM.SetWCO(cfg.WCO)
	l.levelVM.SetWCO(cfg.WCO)

	return l
}

func (l *MeshLeveler) Read() (gcode.Block, error) {
	b, err := l.next()
	if err != nil {
		return nil, err
	}

	oldPos := l.levelVM.WPos()
	err = l.levelVM.Run(b)
	if err != nil {
		return nil, err
	}
	newPos := l.levelVM.WPos()
	if oldPos.Equal(newPos) {
		return b, nil
	}

	// get old and new offset
	// if we don't have one (before or after)
	// then we leave the command as-is
	ok, oldOffset := l.offsetter.OffsetZ(oldPos.X, oldPos.Y)
	if !ok {
		return b, nil
	}
	ok, newOffset := l.offsetter.OffsetZ(newPos.X, newPos.Y)
	if !ok {
		return b, nil
	}
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

	oldPos := l.splitVM.WPos()
	err = l.splitVM.Run(b)
	if err != nil {
		return nil, err
	}
	newPos := l.splitVM.WPos()
	if oldPos.Equal(newPos) {
		return b, nil
	}
	dist := oldPos.DistanceXY(newPos.X, newPos.Y)
	if dist <= l.granularity {
		return b, nil
	}

	// TODO: account for rounding errors past (e.g. beyond .00001)?
	n := int(math.Ceil(dist / l.granularity))
	distPoint := newPos.Sub(oldPos).Div(float64(n))

	if l.splitVM.RelativeMotion() {
		bl := b.Clone()
		bl.SetArg('X', distPoint.X)
		bl.SetArg('Y', distPoint.Y)
		bl.SetArg('Z', distPoint.Z)

		for i := 1; i <= n; i++ {
			l.buf = append(l.buf, bl)
		}
	} else {
		for i := 1; i <= n; i++ {
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
