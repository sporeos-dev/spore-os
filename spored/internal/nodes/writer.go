package nodes

import (
	"bufio"
	"spored/internal/iface"
	"spored/internal/witness"
)

type writer struct {
	writer *bufio.Writer
}

func newWriter(w *bufio.Writer) *writer {
	return &writer{
		writer: w,
	}
}

func (w *writer) WriteRaw(s string) {
	w.writer.WriteString(s)
	w.writer.WriteString("\n")
	w.writer.Flush()
}

func (w *writer) WriteMessage(msg iface.Message) {
	w.writer.WriteString(msg.Wire())
	w.writer.WriteString("\n")
	w.writer.Flush()
	witness.Send(msg)
}

func (w *writer) WriteWitness(msg iface.Message) {
	w.writer.WriteString(msg.Witness())
	w.writer.WriteString("\n")
	w.writer.Flush()
}
