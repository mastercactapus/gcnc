package machine

import (
	"errors"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
)

type ProbeResult struct {
	coord.Point
	Valid bool
}

// ProbeOptions configure a straight z-probe operation.
type ProbeOptions struct {
	ZeroZAxis bool

	// Offset is the offset to use when ZeroZAxis is set.
	Offset float64

	FeedRate  float64
	MaxTravel float64

	// If true, execute a feed hold before probing
	Wait bool
}

// ProbeZ will perform a straigt z-probe from the current location.
func (m *Machine) ProbeZ(opt ProbeOptions) (*ProbeResult, error) {
	if opt.Wait {
		err := m.hold("Attach Z-Probe to spindle.")
		if err != nil {
			return nil, err
		}
	}
	stat := m.CurrentState()
	if stat.Status != "Idle" && stat.Status != "Hold:0" {
		return nil, errors.New("machine not idle")
	}

	m.Adapter.ResetProbes()
	err := m.runBlocks(opt.generate(stat.MPos))
	if err != nil {
		return nil, err
	}
	p := m.Adapter.Probes()
	if len(p) == 0 {
		return nil, errors.New("no probe data returned")
	}

	return &p[0], nil
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
			{W: 'Z', Arg: opt.Offset},
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

// generate will create gcode to do a probe operation that
// handles zeroing the z-axis and returning to the point of origin.
func (opt ProbeOptions) generate(mPos coord.Point) []gcode.Block {
	return opt.probeCommand(opt.ZeroZAxis, mPos.Z)
}
