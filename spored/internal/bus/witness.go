package bus

import "sync"

type witness struct {
	mu sync.RWMutex
	witnesses []inode
}

func newWitness() *witness {
	return &witness {
		witnesses: make([]inode, 0),
	}
}

func (w *witness) close() {}

func (w *witness) register(n inode) {
	w.mu.Lock()
	defer w.mu.Unlock()
	
	if !n.IsWitness() {
		return
	}

	w.witnesses = append(w.witnesses, n)
}

func (w *witness) unregister(n inode) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for i, el := range w.witnesses {
		if el == n {
			w.witnesses[i] = w.witnesses[len(w.witnesses)-1]
			w.witnesses = w.witnesses[:len(w.witnesses)-1]
			return
		}
	}
}

func (w *witness) dispatch(t WitnessType, msg string, n inode) {
	
	msgOut := buildWitnessMessage(t, msg, n)
	
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, el := range w.witnesses {
		el.SendRaw(msgOut)
	}
}
