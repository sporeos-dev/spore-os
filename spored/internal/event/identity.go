package event

import (
	"slices"
	"spored/internal/base"
	"sync"
)

type IdentityEvent[P comparable] struct {
	mu sync.RWMutex
	subscribers []P
	notify func(P, string)
}

// create event
//
// type ISomethingHappens interface {
//   OnSomethingHappens(id string)
// }

// construct event
//
// somethingHappens := event.Identity(
//   func(sub ISomethingHappens, id string) {
//     sub.OnSomethingHappens(id)
//   },
// )

func Identity[P comparable](notify func(P, string)) *IdentityEvent[P] {
	return &IdentityEvent[P]{notify: notify, subscribers: []P{}}
}

func (i *IdentityEvent[P]) Sub(base base.IBase) {
	sub, ok := base.(P)
	if !ok {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	if slices.Contains(i.subscribers, sub) {
		return
	}
	i.subscribers = append(i.subscribers, sub)
}

func (i *IdentityEvent[P]) Unsub(base base.IBase) {
	sub, ok := base.(P)
	if !ok {
		return
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	for j, s := range i.subscribers {
		if s == sub {
			i.subscribers = append(i.subscribers[:j], i.subscribers[j+1:]...)
			break
		}
	}
}

func (i *IdentityEvent[P]) Notify(id string) {
	i.mu.RLock()
	subs := append([]P(nil), i.subscribers...)
	i.mu.RUnlock()

	for _, s := range subs {
		i.notify(s, id)
	}
}
