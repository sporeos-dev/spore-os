package status

import (
	"sync"
)

type Estatus string
const (
	Unverified Estatus = "Unverified"
	RequiresUserSpace Estatus = "RequiresUserSpace"
	Verified Estatus = "Verified"

	Missing Estatus = "Missing"
	FailedChecksum Estatus = "FailedChecksum"
)

type Status struct {
	mu sync.RWMutex
	value Estatus
}

func New() Status {
	return Status{
		value: Unverified,
	}
}

func (s *Status) Get() Estatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.value
}

func (s *Status) Set(newStatus Estatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value = newStatus
}

func (s *Status) String() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return string(s.value)
}
