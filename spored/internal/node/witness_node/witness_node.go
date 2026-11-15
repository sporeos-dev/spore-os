package witness_node

import (
	"fmt"
	"slices"
	"spored/internal/base"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	"spored/internal/manifest"
	"sync"
)

var _ witness_event.IWitnessHandler = (*WitnessHandler)(nil)

type iwriter interface {
	WriteRaw(s string)
}

type WitnessHandler struct {
	*base.Base
	mu sync.RWMutex
	manifest *manifest.Manifest
	writer iwriter
}

func New() *WitnessHandler {
	return &WitnessHandler{
		Base: base.New().SetName("witness-handler"),
	}
}

func (w *WitnessHandler) Open(manifest *manifest.Manifest) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.manifest = manifest
	w.Base.SetId(manifest.ID)

	if slices.Contains(w.manifest.Roles, "witness") {
		event.Subscribe(w)
	}
}

func (w *WitnessHandler) Connect(writer iwriter) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer = writer
}

func (w *WitnessHandler) Disconnect() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer = nil
}

func (w *WitnessHandler) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	event.Unsubscribe(w)
	w.manifest = nil
	w.writer = nil
}

func (w *WitnessHandler) OnWitness(ctx witness_event.IWitnessContext) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.writer == nil {
		return
	}

	var prep string
	switch ctx.Type() {
	case witness_event.WTypeIncoming:
		prep = "from"
	case witness_event.WTypeNode:
		prep = "from"
	case witness_event.WTypeOutgoing:
		prep = "to"
	default:
		prep = "at"
	}

	raw := fmt.Sprintf(
		`witness body='%s' %s spore.time=%d %s=%s`, 
		ctx.Message(), 
		string(ctx.Type()),
		ctx.Time(),
		prep,
		ctx.Origin())
	w.writer.WriteRaw(raw)
}
