package meshlevel

type ZOffsetter interface {
	OffsetZ(x, y float64) (bool, float64)
}

type dummyOffsetter struct {
}

func (dummyOffsetter) OffsetZ(x, y float64) (bool, float64) {
	return false, 0
}
