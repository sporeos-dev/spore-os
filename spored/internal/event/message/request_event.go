package message_event

type IRequestContext interface {
	Message() string
	Origin() string
	Command() string
	Node() string
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
}

func Request(msg, origin, cmd, node string) IRequestContext {
	return &request{
		message: msg, 
		origin: origin, 
		command: cmd,
		node: node,
	}
}

func (r *request) Message() string { 
	return r.message 
}

func (r *request) Origin() string { 
	return r.origin 
}

func (r *request) Command() string { 
	return r.command 
}

func (r *request) Node() string { 
	return r.node 
}
