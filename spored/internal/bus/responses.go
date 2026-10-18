// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"spored/internal/iface"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"sync"

	"time"
)

const responseHandleTimeout = 10 * time.Second

type result struct {
	nodeid string
	expiresAt time.Time
}

type responses struct {
	mu sync.RWMutex
	responses map[string]result
	nodes map[string]INode
}

func newResponses() *responses {
	return &responses {
		responses: make(map[string]result),
		nodes: make(map[string]INode),
	}
}

func (r *responses) close() {}

func (r *responses) register(n INode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nodes[n.Id()] = n
}

func (r *responses) unregister(n INode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.nodes, n.Id())
}

func (r *responses) request(msg iface.Message) *error.Error {
	r.mu.Lock()
	defer r.mu.Unlock()

	handle := msg.Handle()
	nodeid := msg.Cast()

	res, ok := r.responses[handle]
	if ok && time.Now().Before(res.expiresAt) {
		return error.New(
			error.HandleInUse,
			error.Bus,
			"request handle already in use",
			out.Pair("handle", handle)).
			WithMessage(msg)
	}

	r.responses[handle] = result{
		nodeid: nodeid,
		expiresAt: time.Now().Add(responseHandleTimeout),
	}

	return nil
}

func (r *responses) response(msg iface.Message) *error.Error {
	r.mu.Lock()

	handle := msg.Handle()
	if handle == "n/a" {
		return error.New(
			error.Missing,
			error.Bus,
			"missing handle");
	}
	
	res, ok := r.responses[handle]
	if !ok {
		r.mu.Unlock()
		return error.New(
			error.Missing,
			error.Bus,
			"handle not mapped",
			out.Pair("handle", handle)).
			WithMessage(msg)
	}
	delete(r.responses, handle)
	
	nodeid := res.nodeid
	node, ok := r.nodes[nodeid]
	if !ok {
		r.mu.Unlock()
		return error.New(
			error.Missing,
			error.Bus,
			"node not found",
			out.Pair("handle", handle),
			out.Pair("nodeid", nodeid)).
			WithMessage(msg)
	}

	r.mu.Unlock()
	node.Receive(msg)
	return nil
}
