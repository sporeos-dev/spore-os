package nodes

import "sync/atomic"

type estatus int
const (
	Unverified estatus = iota
	RequiresUserSpace 
	Verified

	Missing
	FailedChecksum
)

type status struct {
	atomic.Int32
}

func newStatus() status {
	return status{
		Int32: atomic.Int32{}, // 0 value initialization --> Unverified
	}
}

func (s *status) get() estatus {
	return estatus(s.Load())
}

func (s *status) set(newStatus estatus) {
	s.Store(int32(newStatus))
}
