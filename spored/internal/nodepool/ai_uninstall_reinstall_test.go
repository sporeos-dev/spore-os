// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	node_event "spored/internal/event/node"
	registry_event "spored/internal/event/registry"
	sync_event "spored/internal/event/sync"
	"spored/internal/manifest"
	reg_el "spored/internal/registry/element"
	"testing"
	"time"
)

type lifecycleRegistry struct {
	*base.Base
	withdraw bool
	removed  string
}

func (r *lifecycleRegistry) OnNodeUninstallation(id string) {
	r.removed = id
	if r.withdraw {
		event.Dispatch().RegistryWithdrawal.Notify(registry_event.New(&reg_el.Element{ID: id}))
	}
}

func (r *lifecycleRegistry) OnNodeInstallation(ctx node_event.INodeInstallationContext) {
	el := reg_el.ElementFromManifest(ctx.Manifest())
	event.Dispatch().RegistryAnnouncement.Notify(registry_event.New(el))
	ctx.SetInstalled()
}

func setupLifecycleRegistry(t *testing.T, withdraw bool) (*Pool, *lifecycleRegistry) {
	t.Helper()
	event.Init()
	p := New()
	p.Open()
	registry := &lifecycleRegistry{Base: base.New(), withdraw: withdraw}
	event.Subscribe(registry)
	t.Cleanup(func() {
		event.Unsubscribe(registry)
		p.Close()
	})
	return p, registry
}

func TestUninstallResolvesNodeSuffixToRegisteredID(t *testing.T) {
	const id = "dev.sporeos.foundry.alias"
	for _, name := range []string{"alias", "foundry.alias", "sporeos.foundry.alias", id} {
		t.Run(name, func(t *testing.T) {
			p, registry := setupLifecycleRegistry(t, true)
			p.nodes[id] = lifecycleNode(t, id, manifest.Hyphae, filepath.Join(t.TempDir(), "binary"))

			if !p.nodeUninstall(name) {
				t.Fatal("uninstall failed")
			}
			if registry.removed != id || len(p.nodes) != 0 {
				t.Fatalf("removed=%q, nodes=%v", registry.removed, p.nodes)
			}
		})
	}
}

func TestUninstallAndReinstallRejectAmbiguousSuffix(t *testing.T) {
	for _, reinstall := range []bool{false, true} {
		t.Run(fmt.Sprintf("reinstall=%t", reinstall), func(t *testing.T) {
			p, registry := setupLifecycleRegistry(t, true)
			for _, id := range []string{"com.first.alias", "com.second.alias"} {
				p.nodes[id] = lifecycleNode(t, id, manifest.Hyphae, filepath.Join(t.TempDir(), "binary"))
			}
			var ok bool
			if reinstall {
				ok = p.nodeReinstall("alias")
			} else {
				ok = p.nodeUninstall("alias")
			}
			if ok || registry.removed != "" || len(p.nodes) != 2 {
				t.Fatalf("ambiguous request succeeded or modified registry: ok=%t removed=%q", ok, registry.removed)
			}
		})
	}
}

func TestUninstallReportsRegistryFailure(t *testing.T) {
	p, registry := setupLifecycleRegistry(t, false)
	const id = "dev.sporeos.foundry.alias"
	p.nodes[id] = lifecycleNode(t, id, manifest.Hyphae, filepath.Join(t.TempDir(), "binary"))

	if p.nodeUninstall("alias") || registry.removed != id || p.nodes[id] == nil {
		t.Fatal("failed registry withdrawal was reported as success or removed the node")
	}
}

func TestUninstallDoesNotRemoveNodeWhenKillFails(t *testing.T) {
	p, registry := setupLifecycleRegistry(t, true)
	const id = "dev.sporeos.foundry.alias"
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	single := lifecycleNode(t, id, manifest.Hyphae, executable)
	p.nodes[id] = single

	witness := &lifecycleWitness{Base: base.New(), disconnected: make(chan struct{}, 1)}
	event.Subscribe(witness)
	t.Cleanup(func() { event.Unsubscribe(witness) })
	socketDir, err := os.MkdirTemp("", "spore-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Remove(socketDir); err != nil {
			t.Errorf("remove socket directory: %v", err)
		}
	})
	listener, err := net.Listen("unix", filepath.Join(socketDir, "node.sock"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	client, err := net.Dial("unix", listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Close() })
	conn, err := listener.Accept()
	if err != nil {
		t.Fatal(err)
	}
	single.HandleConnection(conn, bufio.NewReader(conn), bufio.NewWriter(conn))
	if err := client.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if reply, err := bufio.NewReader(client).ReadString('\n'); err != nil || reply != "OK\n" {
		t.Fatalf("handshake reply = %q, err = %v", reply, err)
	}

	handler := &lifecycleHyphae{
		Base: base.New(),
		handle: func(ctx sync_event.ISynchronizedRequestContext) {
			req, ok := cparser.ParseAndValidate(ctx.Message())
			if !ok || req.Command != "dev.sporeos.HYPHAE.hkill" {
				t.Fatalf("unexpected request: %q", ctx.Message())
			}
			raw := fmt.Sprintf("~%s:%s error code=Generic.Hyphae what=failed", req.Handle, req.Command)
			ctx.Respond(message_event.Response(raw, "dev.sporeos.HYPHAE", req.Handle, ctx.Origin()))
		},
	}
	event.Subscribe(handler)
	t.Cleanup(func() { event.Unsubscribe(handler) })
	if p.nodeUninstall("alias") || registry.removed != "" || p.nodes[id] != single {
		t.Fatal("node was removed after kill failed")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-witness.disconnected:
	case <-time.After(time.Second):
		t.Fatal("node did not report disconnect")
	}
}

func TestReinstallUsesResolvedNodeAndRegisteredManifest(t *testing.T) {
	p, registry := setupLifecycleRegistry(t, true)
	const id = "dev.sporeos.foundry.alias"
	p.nodes[id] = lifecycleNode(t, id, manifest.Hyphae, filepath.Join(t.TempDir(), "binary"))
	path := p.nodes[id].Manifest().Path
	confirm := &lifecycleHyphae{
		Base: base.New(),
		handle: func(ctx sync_event.ISynchronizedRequestContext) {
			req, ok := cparser.ParseAndValidate(ctx.Message())
			if !ok || req.Command != "dev.sporeos.shell.confirm" {
				t.Fatalf("unexpected request: %q", ctx.Message())
			}
			raw := fmt.Sprintf("~%s:%s ok yes", req.Handle, req.Command)
			ctx.Respond(message_event.Response(raw, "dev.sporeos.shell", req.Handle, ctx.Origin()))
		},
	}
	event.Subscribe(confirm)
	t.Cleanup(func() { event.Unsubscribe(confirm) })

	if !p.nodeReinstall("sporeos.foundry.alias") {
		t.Fatal("reinstall failed")
	}
	if registry.removed != id || p.nodes[id] == nil || p.nodes[id].Manifest().Path != path {
		t.Fatalf("reinstall used wrong node or manifest: removed=%q", registry.removed)
	}
	p.nodes[id].Close()
}
