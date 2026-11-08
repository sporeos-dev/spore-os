// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"bytes"
	"spored/internal/base"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	"spored/internal/message"
	"strings"
	"sync"
	"testing"
	"time"
)

// noDeadlineConn is a no-op deadlineSetter for tests that don't use a real socket.
type noDeadlineConn struct{}

func (noDeadlineConn) SetWriteDeadline(t time.Time) error { return nil }

type writerWitnessSubscriber struct {
	*base.Base
	events chan witness_event.IWitnessContext
}

func (s *writerWitnessSubscriber) OnWitness(ctx witness_event.IWitnessContext) {
	s.events <- ctx
}

func TestWriterEmitsOutgoingWitnessEvent(t *testing.T) {
	subscriber := &writerWitnessSubscriber{
		Base:   base.New(),
		events: make(chan witness_event.IWitnessContext, 1),
	}
	event.Subscribe(subscriber)
	defer event.Unsubscribe(subscriber)

	var buf bytes.Buffer
	writer := newWriter(noDeadlineConn{}, bufio.NewWriter(&buf), "test.node")
	msg := message.Signal("handle")
	writer.WriteMessage(msg)

	select {
	case ctx := <-subscriber.events:
		if ctx.Type() != witness_event.WTypeOutgoing || ctx.Origin() != "test.node" || ctx.Message() != msg.Wire() {
			t.Errorf("unexpected outgoing witness event: type=%q origin=%q message=%q", ctx.Type(), ctx.Origin(), ctx.Message())
		}
	default:
		t.Fatal("expected outgoing witness event")
	}
}

// TestWriterConcurrentWritesDoNotInterleave guards against the writer.mu
// removal: without serialization, concurrent WriteMessage calls can
// interleave partial writes onto the shared bufio.Writer and corrupt the
// wire protocol (see the "failed to request" / blank-line bug).
func TestWriterConcurrentWritesDoNotInterleave(t *testing.T) {
	var buf bytes.Buffer
	w := newWriter(noDeadlineConn{}, bufio.NewWriter(&buf), "test.node")

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
