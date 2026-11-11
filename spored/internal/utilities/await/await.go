// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package await

import (
	message_event "spored/internal/event/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"sync"
	"time"
)

type Pending struct {
	module error.Module
	mu sync.Mutex
	channels map[string]chan message_event.IResponseContext
	timeout time.Duration
}

func New(module error.Module) *Pending {
	return &Pending{
		module: module,
		channels: make(map[string]chan message_event.IResponseContext),
		timeout: 0,
	}
}

func (p *Pending) WithTimeout(timeout time.Duration) *Pending {
	p.timeout = timeout
	return p
}
func (p *Pending) Await(handle string) chan message_event.IResponseContext {
	ch := make(chan message_event.IResponseContext, 1)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.channels[handle] = ch
	return ch
}

func (p *Pending) Delete(handle string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.channels, handle)
}
func (p *Pending) WaitFor(handle string, ch chan message_event.IResponseContext) (message_event.IResponseContext, bool) {
	return p.WaitForTimeout(handle, ch, nil)
}

func (p *Pending) WaitForTimeout(handle string, ch chan message_event.IResponseContext, override *time.Duration) (message_event.IResponseContext, bool) {
	timeout := p.timeout
	if override != nil {
		timeout = *override
	}

	if timeout <= 0 {
		msg := <-ch
		return msg, true
	}

	select {
	case msg := <-ch:
		return msg, true
	case <-time.After(timeout):
		p.mu.Lock()
		defer p.mu.Unlock()
		delete(p.channels, handle)
		witness.Error(error.Timeout, p.module, "wait timed out", out.Pair("handle", handle))
		return nil, false
	}
}

func (p *Pending) Receive(msg message_event.IResponseContext) {
	handle := msg.Handle()

	p.mu.Lock()
	defer p.mu.Unlock()
	
	ch, ok := p.channels[handle]
	if ok {
		delete(p.channels, handle)
	}
	
	if ok {
		ch <- msg
	} else {
		// no waiter left for this handle (already timed out or never registered) - message would otherwise vanish silently
		witness.Error(error.Missing, p.module, "dropped response for unknown or expired handle", out.Pair("handle", handle))
	}
}

func (p *Pending) Resolve(handle string) *error.Error {
	p.mu.Lock()
	defer p.mu.Unlock()
	ch, ok := p.channels[handle]
	if ok {
		delete(p.channels, handle)
	}

	if ok {
		signal := message_event.Response("resolution signal", "n/a", handle, "n/a")
		ch <- signal
	}

	return nil
}

func (p *Pending) Has(handle string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.channels[handle]
	return ok
}
