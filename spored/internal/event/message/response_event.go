package message_event

import (
	"fmt"
	"spored/internal/utilities/error"
)

type IResponseContext interface {
	Message() string
	Origin() string
	Handle() string
	Destination() string
	Handled()
}

type IResponseHandler interface {
	OnResponse(ctx IResponseContext)
	Ids() []string
}

type response struct {
	message string
	origin string
	handle string
	destination string
	handled bool
}

func Response(msg, origin, handle, dest string) IResponseContext {
	return &response{
		message: msg,
		origin: origin,
		handle: handle,
		destination: dest,
		handled: false,
	}
}

func ResponseError(code error.Code, module error.Module, what, handle, command, dest string) IResponseContext {
	return &response {
		message: fmt.Sprintf("~%s:%s error code=%s.%s what=%s", handle, command, string(code), string(module), what),
		origin: "dev.sporeos.SPORE",
		handle: handle,
		destination: dest,
	}
}

func (r *response) Message() string { return r.message }
func (r *response) Origin() string { return r.origin }
func (r *response) Handle() string { return r.handle }
func (r *response) Destination() string { return r.destination }
func (r *response) Handled() { r.handled = true }

func (r *response) WasHandled() bool { return r.handled }