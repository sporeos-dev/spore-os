package witness_node

import (
	"fmt"
	"log/slog"
	"slices"
	"spored/internal/base"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	"spored/internal/manifest"
	"sync"
	"sync/atomic"
)

var _ witness_event.IWitnessHandler = (*WitnessHandler)(nil)

type iwriter interface {
	WriteRaw(s string)
}

type witnessSession struct {
	writer iwriter
	queue chan witness_event.IWitnessContext
	done chan struct{}
}


func (s *witnessSession) run() {
    for {
        select {
        case <-s.done:
            return

        case ctx := <-s.queue:
            select {
            case <-s.done:
                return
            default:
            }

            var prep string
            switch ctx.Type() {
            case witness_event.WTypeIncoming, witness_event.WTypeNode:
                prep = "from"
            case witness_event.WTypeOutgoing:
                prep = "to"
            default:
                prep = "at"
            }

            raw := fmt.Sprintf(
                `witness body='%s' %s spore.time=%d %s=%s`,
                ctx.Message(), 
				ctx.Type(), 
				ctx.Time(), 
				prep, 
				ctx.Origin(),
            )
            s.writer.WriteRaw(raw)
        }
    }
}

//
//
// witness 
// handler
//

type WitnessHandler struct {
	*base.Base
	mu sync.RWMutex
	manifest *manifest.Manifest
	session *witnessSession
	dropped atomic.Uint64
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

	if w.session != nil {
		close(w.session.done)
	}

	w.session = &witnessSession{
		writer: writer,
		queue: make(chan witness_event.IWitnessContext, 128),
		done: make(chan struct{}),
	}
	go w.session.run()
}

func (w *WitnessHandler) Disconnect() {
	w.mu.Lock()
	if w.session != nil {
		if w.session != nil {
			close(w.session.done)
			w.session = nil
		}
	}
	dropped := w.dropped.Swap(0)
	w.mu.Unlock()

	if dropped > 0 {
		slog.Warn("witness messages dropped", "count", dropped)
	}
}

func (w *WitnessHandler) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()

	event.Unsubscribe(w)
	w.manifest = nil
}

func (w *WitnessHandler) OnWitness(ctx witness_event.IWitnessContext) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.session == nil {
		return
	}

	select {
	case w.session.queue <- ctx:
	default:
		w.dropped.Add(1)
	}
}
