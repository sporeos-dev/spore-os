package nodes

import (
	"bufio"
	"spored/internal/iface"
	"spored/internal/witness"
	"sync"
)

type writer struct {
	mu     sync.Mutex
	writer *bufio.Writer
}

func newWriter(w *bufio.Writer) *writer {
	return &writer{
		writer: w,
	}
}

func (w *writer) WriteRaw(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer.WriteString(s)
	w.writer.WriteString("\n")
	w.writer.Flush()
}

func (w *writer) WriteMessage(msg iface.Message) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer.WriteString(msg.Wire())
	w.writer.WriteString("\n")
	w.writer.Flush()
	witness.Send(msg)
}

func (w *writer) WriteWitness(msg iface.Message) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer.WriteString(msg.Witness())
	w.writer.WriteString("\n")
	w.writer.Flush()
}
