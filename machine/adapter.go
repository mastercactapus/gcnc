package machine

import "io"

// An Adapter represents the minimal CNC machine interface.
type Adapter interface {
	Probes() []ProbeResult
	ResetProbes()

	State() chan State
	CurrentState() State

	WriteByte(byte) error
	Write([]byte) (int, error)
	ReadFrom(io.Reader) (int64, error)
}
