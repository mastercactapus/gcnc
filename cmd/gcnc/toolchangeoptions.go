package main

import (
	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
)

type ToolChangeOptions struct {
	ChangePos    coord.Point
	ProbePos     coord.Point
	FeedRate     float64
	MaxTravel    float64
	TravelHeight float64

	LastToolPos *coord.Point
}

func generateGoTo(travelZ float64, pos coord.Point) []gcode.Block {
	return []gcode.Block{
		{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'Z', Arg: travelZ},
		},
		{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'X', Arg: pos.X},
			{W: 'Y', Arg: pos.Y},
		},
		{
			{W: 'G', Arg: 53},
			{W: 'G', Arg: 0},
			{W: 'Z', Arg: pos.Z},
		},
	}
}

func (opt ToolChangeOptions) GenerateGoToChange() []gcode.Block {
	return generateGoTo(opt.TravelHeight, opt.ChangePos)
}
func (opt ToolChangeOptions) GenerateGoToProbe() []gcode.Block {
	return generateGoTo(opt.TravelHeight, opt.ProbePos)
}
func (opt ToolChangeOptions) GenerateProbe() []gcode.Block {
	var p ProbeOptions
	p.FeedRate = opt.FeedRate
	p.MaxTravel = opt.MaxTravel
	return p.Generate(opt.ProbePos)
}
