package request_node

import (
	"fmt"
	"spored/internal/base"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"spored/internal/manifest"
	"sync"
)

type writer interface {
	WriteRaw(s string)
}

type RequestHandler struct {
	*base.Base
	mu sync.RWMutex
	manifest *manifest.Manifest
	writer writer
	api []string
}

func New() *RequestHandler {
	return &RequestHandler{
		Base: base.New().SetName("request-handler"),
		api: []string{},
	}
}

func (r *RequestHandler) Open(manifest *manifest.Manifest, writer writer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.manifest = manifest
	r.writer = writer
	r.Base.SetId(manifest.ID)
	event.Subscribe(r)

	for _, command := range r.manifest.Api {
		fqName := r.manifest.ID + "." + command.Name
		r.api = append(r.api, fqName)
	}
}

func (r *RequestHandler) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	event.Unsubscribe(r)
	r.manifest = nil
	r.writer = nil
}

func (r *RequestHandler) OnRequest(ctx message_event.IRequestContext) {
	r.mu.Lock()
	defer r.mu.Unlock()

	raw := fmt.Sprintf(
		`%s cast=%s`,
		ctx.Message(),
		ctx.Origin())
	r.writer.WriteRaw(raw)
}

func (r *RequestHandler) Api() []string {
	return r.api
}
