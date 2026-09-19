package nodes

import (
	"bufio"
	"bytes"
	"spored/internal/message"
	"strings"
	"sync"
	"testing"
	"time"
)

// noDeadlineConn is a no-op deadlineSetter for tests that don't use a real socket.
type noDeadlineConn struct{}

func (noDeadlineConn) SetWriteDeadline(t time.Time) error { return nil }

// TestWriterConcurrentWritesDoNotInterleave guards against the writer.mu
// removal: without serialization, concurrent WriteMessage calls can
// interleave partial writes onto the shared bufio.Writer and corrupt the
// wire protocol (see the "failed to request" / blank-line bug).
func TestWriterConcurrentWritesDoNotInterleave(t *testing.T) {
	var buf bytes.Buffer
	w := newWriter(noDeadlineConn{}, bufio.NewWriter(&buf))

	const goroutines = 200
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			w.WriteMessage(message.Signal("handle"))
		}(i)
	}
	wg.Wait()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != goroutines {
		t.Fatalf("expected %d lines, got %d: %q", goroutines, len(lines), buf.String())
	}
	for _, line := range lines {
		if line != lines[0] {
			t.Fatalf("corrupted/interleaved line: %q", line)
		}
	}
}
