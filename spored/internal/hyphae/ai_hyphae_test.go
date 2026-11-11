// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package hyphae

import (
	"fmt"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"sync"
	"testing"
)

// TestHandleUnique guards against the index++ race that let concurrent
// requests (e.g. node.spawn racing a binary hash check) collide on the same
// handle and misroute their responses.
func TestHandleUnique(t *testing.T) {
	const goroutines = 500
	handles := make([]string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			handles[i] = makeHandle()
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool, goroutines)
	for _, handle := range handles {
		if seen[handle] {
			t.Fatalf("duplicate handle generated: %q", handle)
		}
		seen[handle] = true
	}
}

func TestBinaryHashReturnsHashResponseArgument(t *testing.T) {
	event.Init()
	handler := &syncAPITestHandler{
		Base: base.New().SetName("hyphae-test"),
		handle: func(ctx sync_event.ISynchronizedRequestContext) {
			request, ok := cparser.ParseAndValidate(ctx.Message())
			if !ok {
				t.Fatalf("invalid request: %q", ctx.Message())
			}
			if request.Command != "dev.sporeos.HYPHAE.binary.hash" || request.Args["pid"] != "123" || request.Handle == "" {
				t.Fatalf("unexpected request: %+v", request)
			}
			raw := fmt.Sprintf("~%s:%s hash=expected ok", request.Handle, request.Command)
			ctx.Respond(message_event.Response(raw, "dev.sporeos.HYPHAE", request.Handle, ctx.Origin()))
		},
	}
	event.Subscribe(handler)
	t.Cleanup(func() { event.Unsubscribe(handler) })

	hash, ok := BinaryHash(123)
	if !ok {
		t.Fatal("BinaryHash() failed")
	}
	if hash != "expected" {
		t.Fatalf("BinaryHash() = %q, want %q", hash, "expected")
	}
}
