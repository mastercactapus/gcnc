package machine

import (
	"errors"
	"math"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
)

// ProbeGridOptions configure a grid-pattern z-probe operation.
type ProbeGridOptions struct {
	ProbeOptions

	DistanceX, DistanceY float64
	Granularity          float64
}

// ProbeZGrid will perform a grid of straight z-probes.
func (m *Machine) ProbeZGrid(opt ProbeGridOptions) ([]ProbeResult, error) {
	stat := m.CurrentState()
	if stat.Status != "Idle" {
		return nil, errors.New("machine not idle")
	}

	m.ResetProbes()
	err := m.runBlocks(opt.generateGridQuick(stat.MPos))
	if err != nil {
		return nil, err
	}

	startProbes := m.Probes()
	if len(startProbes) == 0 {
		return nil, errors.New("no probe data returned")
	}

	maxZ := startProbes[0].Z
	for _, p := range startProbes[1:] {
		maxZ = math.Max(maxZ, p.Z)
	}
	maxZ += 0.2

	err = m.runBlocks(opt.generateGridSequence(stat.MPos, maxZ))
	if err != nil {
		return nil, err
	}

	return m.Probes(), nil
}

// generateGridQuick creates gcode for a preliminary grid scan.
//
// It scans from the current height for 5 points (corners and center).
func (opt ProbeGridOptions) generateGridQuick(mPos coord.Point) []gcode.Block {
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

// generateGridSequence will generate gcode to do a grid scan by granulartity.
//
// It is intended to be used after GenerateGridQuick is run.
//
// It generates a scan where no two points are farther than granularity apart, returing to the
// provided zHeight during scan, and back to mPos after.
func (opt ProbeGridOptions) generateGridSequence(mPos coord.Point, zHeight float64) []gcode.Block {
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
