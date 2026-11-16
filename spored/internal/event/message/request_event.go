package message_event

import "strings"

type IRequestContext interface {
	Message() string
	Origin() string
	Cast() string
	Command() string
	Node() string
	Handle() string
}

type IRequestHandler interface {
	OnRequest(ctx IRequestContext)
	Api() []string
}

type request struct {
	message string
	origin string
	command string
	node string
	handle string
}

func Request(msg, origin, cmd, node, handle string) IRequestContext {
	return &request{
		message: msg, 
		origin: origin, 
		command: cmd,
		node: node,
		handle: handle,
	}
}

func (r *request) Message() string { 
	return r.message 
}

func (r *request) Origin() string { 
	_, origin, delegated := strings.Cut(r.origin, ":")
	if delegated {
		return origin
	}
	return r.origin
}

func (r *request) Cast() string {
	return r.origin
}

func (r *request) Command() string { 
	return r.command 
}

func (r *request) Node() string { 
	return r.node 
}

func (r *request) Handle() string {
	return r.handle
}
