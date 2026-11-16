package sync_event

import (
	message_event "spored/internal/event/message"
	"time"
)

type ISynchronizedRequestContext interface {
	Message() string
	Origin() string
	Respond(res message_event.IResponseContext)
	Response() message_event.IResponseContext
}

type ISynchronizedRequestHandler interface {
	OnSynchronizedRequest(ctx ISynchronizedRequestContext)
}

type request struct {
	message string
	origin string
	res message_event.IResponseContext
}

type timedRequest struct {
	*request
	timeout *time.Duration
}

func Request(msg, origin string) ISynchronizedRequestContext {
	return &request{
		message: msg,
		origin: origin,
		res: nil,
	}
}

func RequestWithTimeout(msg, origin string, timeout *time.Duration) ISynchronizedRequestContext {
	return &timedRequest{request: &request{message: msg, origin: origin}, timeout: timeout}
}

func (r *timedRequest) TimeoutOverride() *time.Duration {
	return r.timeout
}

func (r *request) Message() string {
	return r.message
}

func (r *request) Origin() string {
	return r.origin
}

func (r *request) Respond(res message_event.IResponseContext) {
	r.res = res
}

func (r *request) Response() message_event.IResponseContext {
	return r.res
}
