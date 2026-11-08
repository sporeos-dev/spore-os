package event

import (
	"slices"
	"spored/internal/base"
	"sync"
)

type ComplexEvent[P comparable, Q any] struct {
	mu sync.RWMutex
	subscribers map[string][]P
	notify func(P, Q)
	list func(P) []string
	key func(Q) string
}

// create event
//
// type ISomethingHappens interface {
//   OnSomethingHappens(object Q)
//   SomethingHappensIds() []string
// }
//
// type ISomethingObject interface {
//   SomethingId() string
// }

// construct event
//
// somethingHappens := event.Complex(
//   func(sub ISomethingHappens, obj Q) {
//     sub.OnSomethingHappens(obj)
//   },
//   func(sub ISomethingHappens) []string {
//     return sub.SomethingHappensIds()
//   },
//   func(obj ISomethingObject) string {
//     return obj.SomethingId()
//   },
// )

func Complex[P comparable, Q any](notify func(P, Q), list func(P) []string, key func(Q) string) *ComplexEvent[P, Q] {
	return &ComplexEvent[P, Q]{ notify: notify, subscribers: map[string][]P{}, list: list, key: key }
}

func (c *ComplexEvent[P, Q]) Sub(base base.IBase) {
	sub, ok := base.(P)
	if !ok {
		return
	}
	list := c.list(sub)

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range list {
		if slices.Contains(c.subscribers[k], sub) {
			continue
		}
		c.subscribers[k] = append(c.subscribers[k], sub)
	}
}

func (c *ComplexEvent[P, Q]) Unsub(base base.IBase) {
	sub, ok := base.(P)
	if !ok {
		return
	}
	list := c.list(sub)

	c.mu.Lock()
	defer c.mu.Unlock()
	for _, k := range list {
		for i, s := range c.subscribers[k] {
			if s != sub {
				continue
			}
			c.subscribers[k] = append(c.subscribers[k][:i], c.subscribers[k][i+1:]...)
			break
		}
	}
}

func (c *ComplexEvent[P, Q]) Notify(obj Q) {
	key := c.key(obj)

	c.mu.RLock()
	subs := append([]P(nil), c.subscribers[key]...)
	c.mu.RUnlock()

	for _, s := range subs {
		c.notify(s, obj)
	}
}