package mycelium

import (
	"spored/internal/base"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"sync"
)

type handleMap struct {
	*base.Base
	mu sync.RWMutex
	handles map[string]string // handle, node
}

func newHandleMap() *handleMap {
	return &handleMap{
		Base: base.New().Set("mycelium", "handle-map"),
		handles: make(map[string]string),
	}
}

func (h *handleMap) mapHandle(handle, node string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	// if already exists
	if _, ok := h.handles[handle]; ok {
		witness.Error(error.Collision, error.Mycelium, "handle exists", out.Pair("handle", handle))
		return false
	}

	h.handles[handle] = node
	return true
}

func (h *handleMap) consumeHandle(handle string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	node, ok := h.handles[handle]
	if ok {
		delete(h.handles, handle)
		return node, ok
	}

	witness.Error(error.Missing, error.Mycelium, "handle missing", out.Pair("handle", handle))
	return "", false
}

