package nodes

import (
	"bufio"
	"log/slog"
	"spored/internal/iface"
	"spored/internal/witness"
	"sync"
	"time"
)

const writeTimeout = 3 * time.Second

// deadlineSetter is the slice of net.Conn the writer needs, kept minimal so
// tests can supply a no-op fake instead of a real socket.
type deadlineSetter interface {
	SetWriteDeadline(t time.Time) error
}

type writer struct {
	mu     sync.Mutex
	conn   deadlineSetter
	writer *bufio.Writer
}

func newWriter(conn deadlineSetter, w *bufio.Writer) *writer {
	return &writer{
		conn:   conn,
		writer: w,
	}
}

// flush bounds the write with a deadline so a stalled peer can't block this writer forever.
func (w *writer) flush() {
	w.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	defer w.conn.SetWriteDeadline(time.Time{})
	if err := w.writer.Flush(); err != nil {
		slog.Warn("write timed out", "error", err)
	}
}

func (w *writer) WriteRaw(s string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer.WriteString(s)
	w.writer.WriteString("\n")
	w.flush()
}

func (w *writer) WriteMessage(msg iface.Message) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer.WriteString(msg.Wire())
	w.writer.WriteString("\n")
	w.flush()
	witness.Send(msg)
}

func (w *writer) WriteWitness(msg iface.Message) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer.WriteString(msg.Witness())
	w.writer.WriteString("\n")
	w.flush()
}
