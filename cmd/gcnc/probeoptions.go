package main

import (
	"math"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
)

type ProbeOptions struct {
	ZeroZAxis bool
	FeedRate  float64
	MaxTravel float64

	// If true, execute a feed hold before probing
	Wait bool
}
type GridOptions struct {
	ProbeOptions

	DistanceX, DistanceY float64
	Granularity          float64
}

// ProbeCommand will return a command to do a Z-probe.
func (opt ProbeOptions) probeCommand(zero bool, lift float64) []gcode.Block {
	b := []gcode.Block{
		{
			{W: 'G', Arg: 91},
			{W: 'G', Arg: 38.2},
			{W: 'Z', Arg: opt.MaxTravel},
			{W: 'F', Arg: opt.FeedRate},
		},
	}
	if zero {
		b = append(b, gcode.Block{
			{W: 'G', Arg: 92},
			{W: 'Z', Arg: 0},
		})
	}
	b = append(b,
		gcode.Block{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'Z', Arg: lift},
		},
	)
	return b
}

// Generate will create gcode to do a probe operation that
// handles zeroing the z-axis and returning to the point of origin.
func (opt ProbeOptions) Generate(mPos coord.Point) []gcode.Block {
	return opt.probeCommand(opt.ZeroZAxis, mPos.Z)
}

// GenerateGridQuick creates gcode for a preliminary grid scan.
//
// It scans from the current height for 5 points (corners and center).
func (opt GridOptions) GenerateGridQuick(mPos coord.Point) []gcode.Block {
	b := opt.probeCommand(opt.ZeroZAxis, mPos.Z)

	probe := func(x, y float64) {
		b = append(b, gcode.Block{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'X', Arg: mPos.X + x},
			{W: 'Y', Arg: mPos.Y + y},
		})
		b = append(b, opt.probeCommand(false, mPos.Z)...)
	}
	probe(0, opt.DistanceY)
	probe(opt.DistanceX/2, opt.DistanceY/2)
	probe(opt.DistanceX, 0)
	probe(opt.DistanceX, opt.DistanceY)
	b = append(b, gcode.Block{
		{W: 'G', Arg: 53},
		{W: 'G', Arg: 0},
		{W: 'X', Arg: mPos.X},
		{W: 'Y', Arg: mPos.Y},
	})

	return b
}

// GenerateGridSequence will generate gcode to do a grid scan by granulartity.
//
// It is intended to be used after GenerateGridQuick is run.
//
// It generates a scan where no two points are farther than granularity apart, returing to the
// provided zHeight during scan, and back to mPos after.
func (opt GridOptions) GenerateGridSequence(mPos coord.Point, zHeight float64) []gcode.Block {
	opt.MaxTravel -= mPos.Z - zHeight

	xyDist := math.Sqrt(opt.Granularity * opt.Granularity / 2)

	xCount := int(math.Ceil(opt.DistanceX / xyDist))
	yCount := int(math.Ceil(opt.DistanceY / xyDist))

	b := []gcode.Block{
		{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'Z', Arg: zHeight},
		},
	}
	probe := func(x, y float64) {
		b = append(b, gcode.Block{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'X', Arg: mPos.X + x},
			{W: 'Y', Arg: mPos.Y + y},
		})
		b = append(b, opt.probeCommand(false, zHeight)...)
	}

	for y := 0; y <= yCount; y++ {
		for x := 0; x <= xCount; x++ {
			xVal := opt.DistanceX / float64(xCount) * float64(x)
			if y%2 != 0 {
				xVal = opt.DistanceX - xVal
			}
			probe(
				xVal,
				opt.DistanceY/float64(yCount)*float64(y),
			)
		}
	}

	b = append(b, gcode.Block{
		{W: 'G', Arg: 53},
		{W: 'G', Arg: 0},
		{W: 'Z', Arg: mPos.Z},
	},
		gcode.Block{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'X', Arg: mPos.X},
			{W: 'Y', Arg: mPos.Y},
		})

	return b
}
