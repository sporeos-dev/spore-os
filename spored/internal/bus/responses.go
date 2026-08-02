package bus

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"sync"
)

type responses struct {
	mu sync.RWMutex
	responses map[string]string
	nodes map[string]INode

}

func newResponses() *responses {
	return &responses {
		responses: make(map[string]string),
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

func (r *responses) request(msg message.Message) *error.Error {
	r.mu.Lock()
	defer r.mu.Unlock()

	handle := msg.Handle()
	nodeid := msg.Cast()

	_, ok := r.responses[handle]
	if ok {
		return error.New(
			error.HandleInUse,
			error.Bus,
			"request handle already in use",
			out.Pair("handle", handle))
	}

	r.responses[handle] = nodeid
	return nil
}

func (r *responses) response(msg message.Message) *error.Error {
	r.mu.Lock()
	defer r.mu.Unlock()

	handle := msg.Handle()
	nodeid, ok := r.responses[handle]
	if !ok {
		return error.New(
			error.Missing,
			error.Bus,
			"handle not mapped",
			out.Pair("handle", handle),
			out.Pair("message", msg.Get()))
	}
	delete(r.responses, handle)
	
	node, ok := r.nodes[nodeid]
	if !ok {
		return error.New(
			error.Missing,
			error.Bus,
			"node not found",
			out.Pair("handle", handle),
			out.Pair("nodeid", nodeid),
			out.Pair("message", msg.Get()))
	}

	node.Receive(msg)
	return nil
}
