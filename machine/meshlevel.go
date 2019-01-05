package machine

import (
	"errors"
	"io"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
	"github.com/mastercactapus/gcnc/meshlevel"
)

func (m *Machine) ReadFromLevel(r io.Reader, granularity float64, points []coord.Point) (int64, error) {
	stat := m.CurrentState()
	if stat.Status != "Idle" {
		return 0, errors.New("machine not idle")
	}

	mesh, err := meshlevel.NewMesh(points)
	if err != nil {
		return 0, err
	}
	cfg := meshlevel.Config{
		ZOffsetter: mesh,

		MPos: stat.MPos,
		WCO:  stat.WCO,

		Granularity: granularity,
		Reader:      gcode.NewParser(r),
	}

	return m.ReadFrom(gcode.NewBuffer(meshlevel.New(cfg)))
}
