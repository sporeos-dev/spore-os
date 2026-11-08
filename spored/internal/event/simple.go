package event

import (
	"slices"
	"spored/internal/base"
	"sync"
)

type SimpleEvent[P comparable, Q any] struct {
	mu sync.RWMutex
	subscribers []P
	notify func(P, Q)
} 

// create event
//
// type ISomethingHappens interface {
//   OnSomethingHappens(object Q)
// }

// construct event
//
// somethingHappens := event.Simple(
//   func(sub ISomethingHappens, obj Q) {
//     sub.OnSomethingHappens(obj)
//   },
// )

func Simple[P comparable, Q any](notify func(P, Q)) *SimpleEvent[P, Q] {
	return &SimpleEvent[P, Q]{ notify: notify, subscribers: []P{} }
}

func (s *SimpleEvent[P, Q]) Sub(base base.IBase) {
	sub, ok := base.(P)
	if !ok {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if slices.Contains(s.subscribers, sub) {
		return
	}
	s.subscribers = append(s.subscribers, sub)
}

func (simp *SimpleEvent[P, Q]) Unsub(base base.IBase) {
	sub, ok := base.(P)
	if !ok {
		return
	}
	
	simp.mu.Lock()
	defer simp.mu.Unlock()
	for i, s := range simp.subscribers {
		if sub == s {
			simp.subscribers = append(simp.subscribers[:i], simp.subscribers[i+1:]...)
			break
		}
	}
}

func (s *SimpleEvent[P, Q]) Notify(obj Q) {
	s.mu.RLock()
	subs := append([]P(nil), s.subscribers...)
	s.mu.RUnlock()

	for _, sub := range subs {
		s.notify(sub, obj)
	}
}

