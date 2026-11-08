package event

import (
	"spored/internal/base"
	"sync"
)

type ievent interface {
	Sub(sub base.IBase)
	Unsub(sub base.IBase)
}

type manager struct {
	mu sync.RWMutex
	events []ievent
	dispatcher *Dispatcher
}

var gmanager *manager
func Init() {
	gmanager = &manager{
		events: []ievent{},
		dispatcher: New(),
	}
}

func Dispatch() *Dispatcher {
	return gmanager.dispatcher
}

func RegisterEvent(ev ievent) {
	gmanager.mu.Lock()
	defer gmanager.mu.Unlock()
	gmanager.events = append(gmanager.events, ev)
}

func UnregisterEvent(ev ievent) {
	gmanager.mu.Lock()
	defer gmanager.mu.Unlock()
	for i, e := range gmanager.events {
		if e == ev {
			gmanager.events = append(gmanager.events[:i], gmanager.events[i+1:]...)
			break
		}
	}
}

func Subscribe(sub base.IBase) {
	gmanager.mu.RLock()
	defer gmanager.mu.RUnlock()
	for _, e := range gmanager.events {
		e.Sub(sub)
	}
	gmanager.dispatcher.Sub(sub)
}

func Unsubscribe(sub base.IBase) {
	gmanager.mu.RLock()
	defer gmanager.mu.RUnlock()
	for _, e := range gmanager.events {
		e.Unsub(sub)
	}
	gmanager.dispatcher.Unsub(sub)
}
