package nodes

import (
	"bufio"
	"spored/internal/message"
	"strings"
)

type writer struct {
	writer *bufio.Writer
}

func newWriter(w *bufio.Writer) *writer {
	return &writer{
		writer: w,
	}
}

// pass nil ibus for witness messages
// to avoid infinite recursion of witnessing
func (w *writer) WriteString(s string, nid string, bus ibus) {
	if !strings.HasSuffix(s, "\n") {
		s += "\n"
	}

	w.writer.WriteString(s)
	w.writer.Flush()
	if bus != nil {
		bus.Witness(message.Outgoing(s, nid))
	}
}