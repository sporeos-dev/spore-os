package request_node

import (
	"fmt"
	"spored/internal/base"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"spored/internal/hyphae"
	"spored/internal/manifest"
	"spored/internal/spore"
	"spored/internal/utilities/await"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"sync"
	"time"
)

var _ message_event.IRequestHandler = (*RequestHandler)(nil)
const pendingTimeout = time.Second * 5

type iwriter interface {
	WriteMessage(s string)
}

type RequestHandler struct {
	*base.Base
	mu sync.RWMutex
	manifest *manifest.Manifest
	writer iwriter
	api []string

	pending *await.Pending
}

func New() *RequestHandler {
	return &RequestHandler{
		Base: base.New().SetName("request-handler"),
		api: []string{},
		pending: await.New(error.Node).WithTimeout(pendingTimeout),
	}
}

func (r *RequestHandler) Open(manifest *manifest.Manifest) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.manifest = manifest
	r.Base.SetId(manifest.ID)

	for _, command := range r.manifest.Api {
		fqName := r.manifest.ID + "." + command.Name
		r.api = append(r.api, fqName)
	}

	event.Subscribe(r)
}

func (r *RequestHandler) Connect(writer iwriter) {
	r.mu.Lock()
	r.writer = writer
	r.mu.Unlock()

	if r.pending.Has("lazy") {
		r.pending.Resolve("lazy")
	}
}

func (r *RequestHandler) Disconnect() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.writer = nil
}

func (r *RequestHandler) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()

	event.Unsubscribe(r)
	r.manifest = nil
	r.writer = nil
	r.api = nil
}

func (r *RequestHandler) OnRequest(ctx message_event.IRequestContext) {
	r.mu.RLock()
	connected := r.writer != nil
	r.mu.RUnlock()

	// handle lazy launch
	if !connected {
		if r.manifest.Launch == manifest.Lazy {
			ch := r.pending.Await("lazy")
			if r.manifest.Namespace == manifest.Hyphae {
				ok := hyphae.Hspawn(r.manifest.GetBinaryPath())
				if !ok {
					witness.Error(error.ConnectionFailure, error.Node, "lazy hyphae spawn failure")
					r.pending.Delete("lazy")
					return
				}
			} else { // manifest.Spore
				ok := spore.NodeSpawn(r.manifest.ID)
				if !ok {
					witness.Error(error.ConnectionFailure, error.Node, "lazy spore spawn failure")
					r.pending.Delete("lazy")
					return
				}
			}
			_, ok := r.pending.WaitFor("lazy", ch)
			if !ok {
				witness.Error(error.Timeout, error.Node, "lazy launch timed out")
				return
			}

		} else {
			witness.Error(error.ConnectionFailure, error.Node, "node not connected", out.Pair("node", r.manifest.ID))
			return
		}
	}

	raw := fmt.Sprintf(
		`%s cast=%s`,
		ctx.Message(),
		ctx.Origin())

	r.mu.RLock()
	defer r.mu.RUnlock()
	r.writer.WriteMessage(raw)
}

func (r *RequestHandler) Api() []string {
	return r.api
}
