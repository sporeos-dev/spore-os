// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package hyphae

import (
	"fmt"
	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/message"
	sporeerror "spored/internal/utilities/error"
	"sync"
	"testing"
)

type testBus struct {
	hyphae *Hyphae
}

func (b *testBus) Register(bus.INode)   {}
func (b *testBus) Unregister(bus.INode) {}

func (b *testBus) Request(request iface.Message) *sporeerror.Error {
	response, ok := message.Response(
		fmt.Sprintf("~%s:dev.sporeos.HYPHAE.binary.hash hash=expected ok", request.Handle()),
		"dev.sporeos.HYPHAE")
	if !ok {
		return sporeerror.New(
			sporeerror.Malformed,
			sporeerror.Hyphae,
			"failed to construct test response")
	}
	return b.hyphae.Receive(response)
}

// TestHandleUnique guards against the index++ race that let concurrent
// requests (e.g. node.spawn racing a binary hash check) collide on the same
// handle and misroute their responses.
func TestHandleUnique(t *testing.T) {
	h := &Hyphae{}

	const goroutines = 500
	handles := make([]string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			handles[i] = h.handle()
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

func TestHashBinaryReturnsHashResponseArgument(t *testing.T) {
	h := New()
	h.bus = &testBus{hyphae: h}

	hash, err := h.HashBinary(123)
	if err != nil {
		t.Fatalf("HashBinary() returned an error: %v", err)
	}
	if hash != "expected" {
		t.Fatalf("HashBinary() = %q, want %q", hash, "expected")
	}
}
