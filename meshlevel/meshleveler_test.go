package meshlevel

import (
	"testing"

	"github.com/mastercactapus/gcnc/coord"
	"github.com/mastercactapus/gcnc/gcode"
	"github.com/stretchr/testify/assert"
)

func TestMeshLeveler(t *testing.T) {

	// probes indicate a rise
	// of 30mm over 100mm or .3mmZ for every 1mm X
	probes := []coord.Point{
		{X: -700, Y: -450, Z: -80},
		{X: -700, Y: -550, Z: -80},

		{X: -600, Y: -450, Z: -50},
		{X: -600, Y: -550, Z: -50},
	}

	mesh, err := NewMesh(probes)
	assert.NoError(t, err)

	// WCO not really set
	// this test has the head floating above the bed
	//
	// we're just checking that moving to the right results
	// in Z being adjusted properly
	cfg := Config{
		ZOffsetter: mesh,

		MPos:        coord.Point{X: -650, Y: -500, Z: -60},
		WCO:         coord.Point{X: -600, Y: -750, Z: -1},
		Granularity: 1,

		Reader: &gcode.BlocksReader{Blocks: gcode.MustParse(`G91 G0 X3`)},
	}

	m := New(cfg)

	b, err := m.Read()
	assert.NoError(t, err)
	assert.Equal(t, "G91G0X1Z0.3", b.String())

	b, err = m.Read()
	assert.NoError(t, err)
	assert.Equal(t, "G91G0X1Z0.3", b.String())

	b, err = m.Read()
	assert.NoError(t, err)
	assert.Equal(t, "G91G0X1Z0.3", b.String())

	b, err = m.Read()
	assert.Error(t, err)

}
