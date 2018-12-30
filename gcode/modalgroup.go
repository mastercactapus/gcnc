package gcode

type ModalGroup byte

const (
	ModalGroupNone = iota
	ModalGroupNonModal
	ModalGroupMotion
	ModalGroupPolar
	ModalGroupPlaneSelection
	ModalGroupDistanceMode
	ModalGroupArcDistanceMode
	ModalGroupFeedRateMode
	ModalGroupUnits
	ModalGroupCutterCompensationMode
	ModalGroupToolLength
	ModalGroupCannedCyclesMode
	ModalGroupCoordinateSystem
	ModalGroupControlMode
	ModalGroupSpindleMode
	ModalGroupLatheDiameterMode
	ModalGroupStopping
	ModalGroupToolChange
	ModalGroupSpindle
	ModalGroupCoolant
	ModalGroupOverride
	ModalGroupFeedRate
)

func (w Word) ModalGroup() ModalGroup {
	if w.W == 'G' {
		switch w.Arg {
		case 4, 10, 28, 30, 53, 92, 92.1, 92.2, 92.3:
			return ModalGroupNonModal
		case 0, 1, 2, 3, 33, 38.2, 38.3, 38.4, 38.5, 73, 76, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89:
			return ModalGroupMotion
		case 15, 16:
			return ModalGroupPolar
		case 17, 18, 19, 17.1, 18.1, 19.1:
			return ModalGroupPlaneSelection
		case 90, 91:
			return ModalGroupDistanceMode
		case 90.1, 91.1:
			return ModalGroupArcDistanceMode
		case 93, 94, 95:
			return ModalGroupFeedRateMode
		case 20, 21:
			return ModalGroupUnits
		case 40, 41, 41.1, 42, 42.1:
			return ModalGroupCutterCompensationMode
		case 43, 43.1, 49:
			return ModalGroupToolLength
		case 98, 99:
			return ModalGroupToolLength
		case 54, 55, 56, 57, 58, 59, 59.1, 59.2, 59.3:
			return ModalGroupCoordinateSystem
		case 61, 61.1, 64:
			return ModalGroupControlMode
		case 96, 97:
			return ModalGroupSpindleMode
		case 7, 8:
			return ModalGroupLatheDiameterMode
		}
	} else if w.W == 'M' {
		switch w.Arg {
		case 0, 1, 2, 30, 60:
			return ModalGroupStopping
		case 6, 61:
			return ModalGroupToolChange
		case 3, 4, 5:
			return ModalGroupSpindle
		case 7, 8, 9:
			return ModalGroupCoolant
		case 48, 49, 50, 51, 52, 53:
			return ModalGroupOverride
		}
	} else if w.W == 'F' {
		return ModalGroupFeedRate
	}

	return ModalGroupNone
}
