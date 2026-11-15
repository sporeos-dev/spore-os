package response_node

import (
	"fmt"
	"spored/internal/base"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"spored/internal/manifest"
	"sync"
)

var _ message_event.IResponseHandler = (*ResponseHandler)(nil)

type iwriter interface {
	WriteMessage(s string)
}

type ResponseHandler struct {
	*base.Base
	mu sync.RWMutex
	manifest *manifest.Manifest
	writer iwriter
}

func New() *ResponseHandler {
	return &ResponseHandler{
		Base: base.New().SetName("response-handler"),
	}
}

func (r *ResponseHandler) Open(manifest *manifest.Manifest) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.manifest = manifest
	r.Base.SetId(manifest.ID)
	event.Subscribe(r)
}

func (r *ResponseHandler) Connect(writer iwriter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writer = writer
}

func (r *ResponseHandler) Disconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writer = nil
}

func (r *ResponseHandler) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	event.Unsubscribe(r)
	r.manifest = nil
	r.writer = nil
}

func (r *ResponseHandler) OnResponse(ctx message_event.IResponseContext) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.writer == nil {
		return
	}

	raw := fmt.Sprintf(
		`%s capture=%s`,
		ctx.Message(),
		ctx.Origin())
	r.writer.WriteMessage(raw)
	
	ctx.Handled()
}

func (r *ResponseHandler) Ids() []string {
	return []string{r.Base.Id()}
}
