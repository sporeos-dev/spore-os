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

type writer interface {
	WriteRaw(s string)
}

type WitnessHandler struct {
	*base.Base
	mu sync.RWMutex
	manifest *manifest.Manifest
	writer writer
}

func New() *WitnessHandler {
	return &WitnessHandler{
		Base: base.New().SetName("witness-handler"),
	}
}

func (w *WitnessHandler) Open(manifest *manifest.Manifest, writer writer) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.manifest = manifest
	w.writer = writer
	w.Base.SetId(manifest.ID)

	if slices.Contains(w.manifest.Roles, "witness") {
		event.Subscribe(w)
	}
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
