// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/registry"
	sporeerror "spored/internal/utilities/error"
	"sync"
	"testing"
)

type testBus struct {
	unregistered bus.INode
}

func (b *testBus) Register(bus.INode) {}

func (b *testBus) Unregister(node bus.INode) {
	b.unregistered = node
}

func (b *testBus) Request(iface.Message) *sporeerror.Error   { return nil }
func (b *testBus) Response(iface.Message) *sporeerror.Error  { return nil }
func (b *testBus) Broadcast(iface.Message) *sporeerror.Error { return nil }
func (b *testBus) Pipe(iface.Message, bus.INode)             {}
func (b *testBus) Witness(iface.Message)                     {}

// TestNodesHandleUnique guards against the index++ race: concurrent Install/
// Uninstall-triggered handle() calls must never produce the same handle,
// otherwise Pending.Await entries silently clobber each other and responses
// get delivered to the wrong waiter.
func TestNodesHandleUnique(t *testing.T) {
	n := &Nodes{}

	const goroutines = 500
	handles := make([]string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			handles[i] = n.handle()
		}(i)
	}
	wg.Wait()

	seen := make(map[string]bool, goroutines)
	for _, h := range handles {
		if seen[h] {
			t.Fatalf("duplicate handle generated: %q", h)
		}
		seen[h] = true
	}
}

func TestNodeCloseUnregistersNode(t *testing.T) {
	testBus := &testBus{}
	node := newNode(&registry.Element{}, &manifest.Manifest{})
	node.bus = testBus

	node.close()

	if testBus.unregistered != node {
		t.Fatal("close() did not unregister the node")
	}
}
