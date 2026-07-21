package bus

import (
	"slices"
	"spored/internal/message"
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

func (w *witness) in(raw string, id string) {
	msg := message.Witness(message.WitnessIncoming, raw, id)

	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, el := range w.witnesses {
		if !el.IsConnected() {
			continue
		}
		el.Receive(msg)
	}
}

func (w *witness) out(raw string, id string) {
	msg := message.Witness(message.WitnessOutgoing, raw, id)

	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, el := range w.witnesses {
		if !el.IsConnected() {
			continue
		}
		el.Receive(msg)
	}
}

func (w *witness) spore(raw string) {
	msg := message.Witness(message.WitnessSpore, raw, "dev.sporeos.SPORE")

	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, el := range w.witnesses {
		if !el.IsConnected() {
			continue
		}
		el.Receive(msg)
	}
}

func (w *witness) node(raw string, id string) {
	msg := message.Witness(message.WitnessNode, raw, id)

	w.mu.RLock()
	defer w.mu.RUnlock()
	for _, el := range w.witnesses {
		if !el.IsConnected() {
			continue
		}
		el.Receive(msg)
	}
}
