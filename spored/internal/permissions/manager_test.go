// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package permissions

import (
	"sync"
	"testing"
)

// TestHandleUnique guards against the index++ race in handle generation.
func TestHandleUnique(t *testing.T) {
	m := &Manager{}

	const goroutines = 500
	handles := make([]string, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			handles[i] = m.handle()
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
