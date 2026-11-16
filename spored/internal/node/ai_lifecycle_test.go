// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package node

import (
	"bufio"
	"bytes"
	"net"
	"os"
	"path/filepath"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	reg_el "spored/internal/registry/element"
	"testing"
	"time"
)

type testDeadlineSetter struct{}

func (testDeadlineSetter) SetWriteDeadline(time.Time) error { return nil }

func TestDisconnectKeepsNodeRequestsAvailable(t *testing.T) {
	event.Init()
	const id = "com.example.node"
	path := filepath.Join(t.TempDir(), "node.manifest.spore.yaml")
	content := "id: " + id + "\napp: node\napi:\n  - name: ping\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	single := New(&reg_el.Element{ID: id, Manifest: path, Binary: "node"})
	t.Cleanup(single.Close)

	conn, peer := net.Pipe()
	single.conn = conn
	single.reader = bufio.NewReader(conn)
	single.writer = newWriter(conn, bufio.NewWriter(conn), id)
	single.requestHandler.Connect(single.writer)
	done := make(chan struct{})
	go func() {
		single.listen()
		close(done)
	}()
	if err := peer.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("node did not disconnect")
	}

	var output bytes.Buffer
	single.requestHandler.Connect(newWriter(testDeadlineSetter{}, bufio.NewWriter(&output), id))
	event.Dispatch().Request.Notify(message_event.Request("ping ~h", "caller", id+".ping", id, "h"))
	if got, want := output.String(), "ping ~h cast=caller\n"; got != want {
		t.Fatalf("forwarded request = %q, want %q", got, want)
	}
}

func TestCloseWithMalformedManifest(t *testing.T) {
	event.Init()
	path := filepath.Join(t.TempDir(), "node.manifest.spore.yaml")
	if err := os.WriteFile(path, []byte("id: ["), 0600); err != nil {
		t.Fatal(err)
	}
	single := New(&reg_el.Element{ID: "com.example.node", Manifest: path})
	single.Close()
}
