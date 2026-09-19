package bus

import (
	"log/slog"
	"slices"
	"spored/internal/iface"
	"sync"
)

type witness struct {
	mu sync.RWMutex
	witnesses []INode
}

func newWitness() *witness {
	return &witness {
		witnesses: make([]INode, 0),
	}
}

func (w *witness) close() {}

func (w *witness) register(n INode) {
	if !n.IsWitness() {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	
	if slices.Contains(w.witnesses, n) {
		return
	}
	w.witnesses = append(w.witnesses, n)
}

func (w *witness) unregister(n INode) {
	if !n.IsWitness() {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if !slices.Contains(w.witnesses, n) {
		return
	}
	for i, el := range w.witnesses {
		if el == n {
			w.witnesses[i] = w.witnesses[len(w.witnesses)-1]
			w.witnesses = w.witnesses[:len(w.witnesses)-1]
			return
		}
	}
}

func (w *witness) witness(msg iface.Message) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	slog.Info(msg.Witness())
	for _, el := range w.witnesses {
		if !el.IsConnected() {
			continue
		}
		el.Witness(msg)
	}
}
