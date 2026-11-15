// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package request_node

import (
	"fmt"
	"path/filepath"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"spored/internal/manifest"
	"testing"
)

type lazyTestWriter struct {
	message string
}

func (w *lazyTestWriter) WriteMessage(message string) { w.message = message }

type lazyTestHyphae struct {
	*base.Base
	handle func(sync_event.ISynchronizedRequestContext)
}

func (h *lazyTestHyphae) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	h.handle(ctx)
}

func TestLazyHyphaeSpawnUsesBinaryPath(t *testing.T) {
	event.Init()
	m := &manifest.Manifest{
		ID:        "com.example.node",
		Path:      filepath.Join(t.TempDir(), "node.manifest.spore.yaml"),
		App:       "example-binary",
		Namespace: manifest.Hyphae,
		Launch:    manifest.Lazy,
	}
	handler := New()
	handler.Open(m)
	t.Cleanup(handler.Close)
	writer := &lazyTestWriter{}
	hyphae := &lazyTestHyphae{
		Base: base.New().SetName("lazy-spawn-test"),
		handle: func(ctx sync_event.ISynchronizedRequestContext) {
			req, ok := cparser.ParseAndValidate(ctx.Message())
			if !ok || req.Command != "dev.sporeos.HYPHAE.hspawn" || req.Args["binary"] != m.GetBinaryPath() {
				t.Fatalf("unexpected spawn request: %q", ctx.Message())
			}
			handler.Connect(writer)
			raw := fmt.Sprintf("~%s:%s ok", req.Handle, req.Command)
			ctx.Respond(message_event.Response(raw, "dev.sporeos.HYPHAE", req.Handle, ctx.Origin()))
		},
	}
	event.Subscribe(hyphae)
	t.Cleanup(func() { event.Unsubscribe(hyphae) })

	handler.OnRequest(message_event.Request("node.command ~h1", "caller", "com.example.node.command", m.ID, "h1"))
	if writer.message != "node.command ~h1 cast=caller" {
		t.Fatalf("forwarded request = %q", writer.message)
	}
}
