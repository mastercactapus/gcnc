package meshlevel

type ZOffsetter interface {
	OffsetZ(x, y float64) (bool, float64)
}
