package event

import (
	"slices"
	"spored/internal/base"
	"sync"
)

type EmptyEvent[P comparable] struct {
	mu sync.RWMutex
	subscribers []P
	notify func(P)
}

// create event
//
// type ISomethingHappens interface {
//   OnSomethingHappens()
// }

// construct event
//
// somethingHappens := event.Empty(
//   func(sub ISomethingHappens) {
//     sub.OnSomethingHappens()
//   },
// )

func Empty[P comparable](notify func(P)) *EmptyEvent[P] {
	return &EmptyEvent[P]{notify: notify, subscribers: []P{}}
}

func (e *EmptyEvent[P]) Sub(base base.IBase) {
	sub, ok := base.(P)
	if !ok {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if slices.Contains(e.subscribers, sub) {
		return
	}
	e.subscribers = append(e.subscribers, sub)
}

func (e *EmptyEvent[P]) Unsub(base base.IBase) {
	sub, ok := base.(P)
	if !ok {
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	for i, s := range e.subscribers {
		if s == sub {
			e.subscribers = append(e.subscribers[:i], e.subscribers[i+1:]...)
			break
		}
	}
}

func (e *EmptyEvent[P]) Notify() {
	e.mu.RLock()
	subs := append([]P(nil), e.subscribers...)
	e.mu.RUnlock()
	
	for _, s := range subs {
		e.notify(s)
	}
}
