package broadcast_node

import (
	"fmt"
	"spored/internal/base"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"spored/internal/manifest"
	"sync"
)

var _ message_event.IBroadcastHandler = (*BroadcastHandler)(nil)

type iwriter interface {
	WriteMessage(s string)
}

type BroadcastHandler struct {
	*base.Base
	mu sync.RWMutex
	manifest *manifest.Manifest
	writer iwriter
}

func New() *BroadcastHandler {
	return &BroadcastHandler{
		Base: base.New().SetName("broadcast-handler"),
	}
}

func (b *BroadcastHandler) Open(manifest *manifest.Manifest) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.manifest = manifest
	b.Base.SetId(manifest.ID)
	event.Subscribe(b)
}

func (b *BroadcastHandler) Connect(writer iwriter) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.writer = writer
}

func (b *BroadcastHandler) Disconnect() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.writer = nil
}

func (b *BroadcastHandler) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	event.Unsubscribe(b)
	b.manifest = nil
	b.writer = nil
}

func (b *BroadcastHandler) OnBroadcast(ctx message_event.IBroadcastContext) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if b.writer == nil {
		return
	}

	raw := fmt.Sprintf(
		`publish %s cast=%s`,
		ctx.Message(),
		ctx.Origin())
	b.writer.WriteMessage(raw)
}

func (b *BroadcastHandler) Ids() []string {
	return []string{b.Base.Id()}
}