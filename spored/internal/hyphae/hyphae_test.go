package hyphae

import (
	"sync"
	"testing"
)

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
