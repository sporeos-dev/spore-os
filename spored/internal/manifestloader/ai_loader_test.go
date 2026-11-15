// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package manifestloader

import (
	"fmt"
	"os"
	"path/filepath"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"spored/internal/manifest"
	"spored/internal/utilities/status"
	"testing"
)

type testNotifier struct {
	announcements int
}

func (n *testNotifier) Announce(*manifest.Manifest) { n.announcements++ }
func (n *testNotifier) Withdraw(*manifest.Manifest) {}

type testHyphae struct {
	*base.Base
	handle func(sync_event.ISynchronizedRequestContext)
}

func (h *testHyphae) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	h.handle(ctx)
}

func TestLoadLocalManifest(t *testing.T) {
	event.Init()
	path := filepath.Join(t.TempDir(), "node.yaml")
	if err := os.WriteFile(path, []byte("id: example.node\n"), 0600); err != nil {
		t.Fatal(err)
	}
	m := manifest.Empty(path, "")
	notifier := &testNotifier{}
	m.Notifier = notifier

	if !Load(m) || !Load(m) {
		t.Fatal("loading local manifest failed")
	}
	if m.ID != "example.node" || m.Launch != manifest.Manual || m.Namespace != manifest.Hyphae {
		t.Fatalf("unexpected manifest: id=%q launch=%q namespace=%q", m.ID, m.Launch, m.Namespace)
	}
	if notifier.announcements != 1 {
		t.Fatalf("announcements = %d, want 1", notifier.announcements)
	}
}

func TestLoadManifestThroughHyphae(t *testing.T) {
	event.Init()
	path := filepath.Join(t.TempDir(), "unavailable.yaml")
	handler := &testHyphae{
		Base: base.New().SetName("manifest-loader-test"),
		handle: func(ctx sync_event.ISynchronizedRequestContext) {
			request, ok := cparser.ParseAndValidate(ctx.Message())
			if !ok {
				t.Fatalf("invalid request: %q", ctx.Message())
			}
			if request.Command != "dev.sporeos.HYPHAE.manifest.read" || request.Args["path"] != path {
				t.Fatalf("unexpected request: %+v", request)
			}
			raw := fmt.Sprintf("~%s:%s ok content=<<\nid: example.user\n>>", request.Handle, request.Command)
			ctx.Respond(message_event.Response(raw, "dev.sporeos.HYPHAE", request.Handle, ctx.Origin()))
		},
	}
	event.Subscribe(handler)
	t.Cleanup(func() { event.Unsubscribe(handler) })

	m := manifest.Empty(path, "")
	if !Load(m) || m.ID != "example.user" {
		t.Fatalf("loading through Hyphae failed: id=%q", m.ID)
	}
}

func TestLoadMalformedManifest(t *testing.T) {
	event.Init()
	path := filepath.Join(t.TempDir(), "malformed.yaml")
	if err := os.WriteFile(path, []byte("id: ["), 0600); err != nil {
		t.Fatal(err)
	}
	m := manifest.Empty(path, "")
	notifier := &testNotifier{}
	m.Notifier = notifier

	if Load(m) {
		t.Fatal("malformed manifest loaded successfully")
	}
	if m.Status.Get() != status.Malformed || notifier.announcements != 0 {
		t.Fatalf("status=%s announcements=%d", m.Status.Get(), notifier.announcements)
	}
}

func TestLoadHyphaeFailureDoesNotAnnounce(t *testing.T) {
	event.Init()
	m := manifest.Empty(filepath.Join(t.TempDir(), "unavailable.yaml"), "")
	notifier := &testNotifier{}
	m.Notifier = notifier

	if Load(m) {
		t.Fatal("manifest loaded without a Hyphae response")
	}
	if notifier.announcements != 0 {
		t.Fatalf("announcements = %d, want 0", notifier.announcements)
	}
}
