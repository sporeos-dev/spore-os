package witness_node

import (
	witness_event "spored/internal/event/witness"
	"strings"
	"testing"
	"time"
)

type testWriter struct {
	messages chan string
}

func (w *testWriter) WriteRaw(raw string) {
	w.messages <- raw
}

func TestConnectAndReconnectWitness(t *testing.T) {
	handler := New()
	t.Cleanup(handler.Disconnect)

	first := &testWriter{messages: make(chan string, 1)}
	handler.Connect(first)
	handler.OnWitness(witness_event.Incoming("first", "node"))
	assertWitnessMessage(t, first, "first")

	second := &testWriter{messages: make(chan string, 1)}
	handler.Connect(second)
	handler.OnWitness(witness_event.Incoming("second", "node"))
	assertWitnessMessage(t, second, "second")
}

func assertWitnessMessage(t *testing.T, writer *testWriter, want string) {
	t.Helper()
	select {
	case raw := <-writer.messages:
		if !strings.Contains(raw, want) {
			t.Fatalf("witness message = %q, want %q", raw, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for witness message")
	}
}
