// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	sync_event "spored/internal/event/sync"
	"spored/internal/manifest"
	"spored/internal/node"
	reg_el "spored/internal/registry/element"
	"strconv"
	"testing"
	"time"
)

type lifecycleHyphae struct {
	*base.Base
	handle func(sync_event.ISynchronizedRequestContext)
}

func (h *lifecycleHyphae) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	h.handle(ctx)
}

type lifecycleMessages struct {
	*base.Base
	messages chan string
}

func (h *lifecycleMessages) OnMessage(ctx message_event.IMessageContext) {
	h.messages <- ctx.Message()
}

func lifecycleNode(t *testing.T, id string, namespace manifest.Namespace, binary string) *node.Single {
	t.Helper()
	path := filepath.Join(t.TempDir(), "node.manifest.spore.yaml")
	content := fmt.Sprintf("id: %s\ntrust: developer\nnamespace: %s\napp: binary\n", id, namespace)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	single := node.New(&reg_el.Element{ID: id, Manifest: path, Binary: binary, Checksum: "developer"})
	t.Cleanup(single.Close)
	return single
}

func TestNodeSpawnThroughHyphae(t *testing.T) {
	for _, namespace := range []manifest.Namespace{manifest.Hyphae, manifest.Any} {
		for _, success := range []bool{true, false} {
			t.Run(fmt.Sprintf("namespace=%s/success=%t", namespace, success), func(t *testing.T) {
				event.Init()
				const id = "com.example.clock"
				binary := filepath.Join(t.TempDir(), "example binary")
				p := New()
				p.nodes[id] = lifecycleNode(t, id, namespace, binary)

				handler := &lifecycleHyphae{
					Base: base.New().SetName("lifecycle-hyphae-test"),
					handle: func(ctx sync_event.ISynchronizedRequestContext) {
						req, ok := cparser.ParseAndValidate(ctx.Message())
						if !ok || req.Command != "dev.sporeos.HYPHAE.hspawn" || req.Args["binary"] != binary {
							t.Fatalf("unexpected Hyphae request: %q", ctx.Message())
						}
						result := "ok"
						if !success {
							result = `error code=Generic.Hyphae what="spawn failed"`
						}
						raw := fmt.Sprintf("~%s:%s %s", req.Handle, req.Command, result)
						ctx.Respond(message_event.Response(raw, "dev.sporeos.HYPHAE", req.Handle, ctx.Origin()))
					},
				}
				event.Subscribe(handler)
				t.Cleanup(func() { event.Unsubscribe(handler) })

				if got := p.nodeSpawn("clock"); got != success {
					t.Fatalf("nodeSpawn() = %t, want %t", got, success)
				}
			})
		}
	}
}

func TestNodeSpawnInSporeSpace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a Unix executable")
	}
	for _, namespace := range []manifest.Namespace{manifest.Spore, manifest.Any} {
		t.Run(string(namespace), func(t *testing.T) {
			event.Init()
			binary := filepath.Join(t.TempDir(), "node")
			if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
				t.Fatal(err)
			}
			const id = "com.example.local"
			p := New()
			p.nodes[id] = lifecycleNode(t, id, namespace, binary)
			if !p.nodeSpawn(id) {
				t.Fatal("nodeSpawn() failed")
			}
		})
	}
}

func TestNodeLifecycleRejectsMissingAndDisconnected(t *testing.T) {
	event.Init()
	p := New()
	if p.nodeSpawn("missing") || p.nodeKill("missing") {
		t.Fatal("missing node accepted")
	}

	const id = "com.example.disconnected"
	p.nodes[id] = lifecycleNode(t, id, manifest.Spore, filepath.Join(t.TempDir(), "binary"))
	if p.nodeKill(id) {
		t.Fatal("disconnected node accepted")
	}
}

func TestNodeKillThroughHyphae(t *testing.T) {
	event.Init()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	const id = "com.example.user"
	p := New()
	single := lifecycleNode(t, id, manifest.Hyphae, executable)
	p.nodes[id] = single

	listener, err := net.Listen("unix", filepath.Join(t.TempDir(), "node.sock"))
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
		Base: base.New().SetName("lifecycle-hkill-test"),
		handle: func(ctx sync_event.ISynchronizedRequestContext) {
			req, ok := cparser.ParseAndValidate(ctx.Message())
			if !ok || req.Command != "dev.sporeos.HYPHAE.hkill" || req.Args["pid"] != strconv.Itoa(os.Getpid()) {
				t.Fatalf("unexpected Hyphae request: %q", ctx.Message())
			}
			raw := fmt.Sprintf("~%s:%s ok", req.Handle, req.Command)
			ctx.Respond(message_event.Response(raw, "dev.sporeos.HYPHAE", req.Handle, ctx.Origin()))
		},
	}
	event.Subscribe(handler)
	t.Cleanup(func() { event.Unsubscribe(handler) })
	if !p.nodeKill(id) {
		t.Fatal("nodeKill() failed")
	}
}

func TestNodeLifecycleErrorsUseMessageEvent(t *testing.T) {
	for _, command := range []string{"dev.sporeos.SPORE.node.spawn", "dev.sporeos.SPORE.node.kill"} {
		t.Run(command, func(t *testing.T) {
			event.Init()
			capture := &lifecycleMessages{
				Base:     base.New().SetName("lifecycle-message-test"),
				messages: make(chan string, 1),
			}
			event.Subscribe(capture)
			t.Cleanup(func() { event.Unsubscribe(capture) })

			p := New()
			p.OnRequest(message_event.Request(command+" node=missing ~h1", "caller", command, "dev.sporeos.SPORE", "h1"))
			select {
			case raw := <-capture.messages:
				response, ok := cparser.ParseAndValidate(raw)
				if !ok || response.Handle != "h1" || response.Command != command || !slices.Contains(response.Flags, "error") {
					t.Fatalf("unexpected error response: %q", raw)
				}
			case <-time.After(time.Second):
				t.Fatal("no error response routed through Mycelium")
			}
		})
	}
}

func TestFindNodeRejectsAmbiguousSuffix(t *testing.T) {
	event.Init()
	p := New()
	p.nodes["com.first.clock"] = &node.Single{}
	p.nodes["com.second.clock"] = &node.Single{}
	if _, ok := p.findNode("clock"); ok {
		t.Fatal("ambiguous suffix accepted")
	}
	if _, ok := p.findNode("com.first.clock"); !ok {
		t.Fatal("exact ID rejected")
	}
}

func TestKillLocal(t *testing.T) {
	event.Init()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(executable, "-test.run=^TestLifecycleChild$")
	cmd.Env = append(os.Environ(), "SPORE_TEST_LIFECYCLE_CHILD=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if cmd.ProcessState == nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})
	if !killLocal("test.child", cmd.Process.Pid) {
		t.Fatal("killLocal() failed")
	}
	if err := cmd.Wait(); err == nil {
		t.Fatal("child exited normally instead of being killed")
	}
}

func TestLifecycleChild(t *testing.T) {
	if os.Getenv("SPORE_TEST_LIFECYCLE_CHILD") == "1" {
		time.Sleep(time.Minute)
	}
}
