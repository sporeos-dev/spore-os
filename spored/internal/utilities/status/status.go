package status

import "sync/atomic"

type Estatus int
const (
	Unverified Estatus = iota
	RequiresUserSpace 
	Verified

	Missing
	FailedChecksum
)

type Status struct {
	atomic.Int32
}

func New() Status {
	return Status{
		Int32: atomic.Int32{}, // 0 value initialization --> Unverified
	}
}

func (s *Status) Get() Estatus {
	return Estatus(s.Load())
}

func (s *Status) Set(newStatus Estatus) {
	s.Store(int32(newStatus))
}
