package await

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"sync"
	"time"
)

type Pending struct {
	module error.Module
	mu sync.Mutex
	channels map[string]chan message.Message
	timeout time.Duration
}

func New(module error.Module) *Pending {
	return &Pending{
		module: module,
		channels: make(map[string]chan message.Message),
		timeout: 0, 
	}
}

func (p *Pending) WithTimeout(timeout time.Duration) *Pending {
	p.timeout = timeout
	return p
}

func (p *Pending) Await(handle string) chan message.Message {
	ch := make(chan message.Message, 1)
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

func (p *Pending) WaitFor(handle string, ch chan message.Message) (message.Message, *error.Error) {
	if p.timeout == 0 {
		msg := <-ch
		return msg, nil
	}
	
	select {
	case msg := <-ch:
		return msg, nil
	case <-time.After(p.timeout):
		p.mu.Lock()
		defer p.mu.Unlock()
		return nil, error.New(
			error.Timeout,
			p.module,
			"timed out pending",
			out.Pair("handle", handle))
	}
}

func (p *Pending) Receive(msg message.Message) *error.Error {
	handle := msg.Handle()

	p.mu.Lock()
	defer p.mu.Unlock()
	
	ch, ok := p.channels[handle]
	if ok {
		delete(p.channels, handle)
	}
	
	if ok {
		ch <- msg
	}

	return nil
}
