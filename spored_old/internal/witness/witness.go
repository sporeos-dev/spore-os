// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package witness

import (
	"fmt"
	"spored/internal/interfaces"
	"sync"
	"time"
)

type Witness struct {
	mu sync.RWMutex
	nodes []interfaces.Node
}

func (w *Witness) Register(node interfaces.Node) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if node.GetManifest().Witness {
		w.nodes = append(w.nodes, node)
	}
}

func (w *Witness) Unregister(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i, node := range w.nodes {
		if node.GetManifest().ID == id {
			w.nodes = append(w.nodes[:i], w.nodes[i+1:]...)
			return
		}
	}
}

func (w* Witness) Incoming(msg string) {
	w.mu.RLock()
	nodes := w.nodes
	w.mu.RUnlock()

	msg = fmt.Sprintf("witness %s spore_incoming spore_time=%d", msg, time.Now().UnixMilli())

	for _, node := range nodes {
		node.SendWitness(msg)
	}
}

func (w* Witness) Outgoing(msg string) {
	w.mu.RLock()
	nodes := w.nodes
	w.mu.RUnlock()

	msg = fmt.Sprintf("witness %s spore_outgoing spore_time=%d", msg, time.Now().UnixMilli())

	for _, node := range nodes {
		node.SendWitness(msg)
	}
}

func (w* Witness) Spore(msg string) {
	w.mu.RLock()
	nodes := w.nodes
	w.mu.RUnlock()

	msg = fmt.Sprintf("witness %s spore_event spore_time=%d", msg, time.Now().UnixMilli())

	for _, node := range nodes {
		node.SendWitness(msg)
	}
}

func (w* Witness) Node(msg string, cast string) {
	w.mu.RLock()
	nodes := w.nodes
	w.mu.RUnlock()

	msg = fmt.Sprintf("witness %s cast=%s spore_node spore_time=%d", msg, cast, time.Now().UnixMilli())

	for _, node := range nodes {
		node.SendWitness(msg)
	}
}