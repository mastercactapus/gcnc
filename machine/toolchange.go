package machine

import (
	"errors"
	"log"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
)

// ToolChangeOptions configure a tool change operation.
type ToolChangeOptions struct {
	ChangePos    coord.Point
	ProbePos     coord.Point
	FeedRate     float64
	MaxTravel    float64
	TravelHeight float64

	LastToolPos *coord.Point
}

// ToolChange will immediatly start a tool change operation.
func (m *Machine) ToolChange(opt ToolChangeOptions) error {
	stat := m.CurrentState()
	if stat.Status != "Idle" {
		return errors.New("machine not idle")
	}

	if opt.LastToolPos == nil {
		// get current tool first
		p, err := m.toolProbe(opt, false, 0)
		if err != nil {
			return err
		}
		opt.LastToolPos = &p.Point
		err = m.hold("Probe complete, remove Z-Probe.")
		if err != nil {
			return err
		}
	}

	err := m.runBlocks(generateGoTo(opt.TravelHeight, opt.ChangePos))
	if err != nil {
		return err
	}

	m.hold("Perform tool change.")
	if err != nil {
		return err
	}

	lastToolWPos := opt.LastToolPos.Sub(stat.WCO)
	p, err := m.toolProbe(opt, true, lastToolWPos.Z)
	if err != nil {
		return err
	}

	diff := opt.LastToolPos.Z - p.Z
	stat.MPos.Z -= diff
	log.Println("Adjusting Z-offset by:", diff)

	m.hold("Probe complete, remove Z-Probe.")
	if err != nil {
		return err
	}

	err = m.runBlocks(generateGoTo(opt.TravelHeight, stat.MPos))
	if err != nil {
		return err
	}

	return nil
}

func (m *Machine) toolProbe(opt ToolChangeOptions, zero bool, offset float64) (*ProbeResult, error) {
	err := m.runBlocks(generateGoTo(opt.TravelHeight, opt.ProbePos))
	if err != nil {
		return nil, err
	}

	p, err := m.ProbeZ(ProbeOptions{
		Wait:      true,
		MaxTravel: opt.MaxTravel,
		FeedRate:  opt.FeedRate,
		ZeroZAxis: zero,
		Offset:    offset,
	})
	if err != nil {
		return nil, err
	}
	if !p.Valid {
		return nil, errors.New("tool probe failed")
	}
	return p, nil
}

func (opt ToolChangeOptions) generateGoToChange() []gcode.Block {
	return generateGoTo(opt.TravelHeight, opt.ChangePos)
}
func (opt ToolChangeOptions) generateGoToProbe() []gcode.Block {
	return generateGoTo(opt.TravelHeight, opt.ProbePos)
}
func (opt ToolChangeOptions) generateProbe() []gcode.Block {
	var p ProbeOptions
	p.FeedRate = opt.FeedRate
	p.MaxTravel = opt.MaxTravel
	return p.generate(opt.ProbePos)
}
