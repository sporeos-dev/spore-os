// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"sync"
	"testing"
)

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
