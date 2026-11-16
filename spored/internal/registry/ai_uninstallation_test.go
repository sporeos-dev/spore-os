// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	registry_event "spored/internal/event/registry"
	sync_event "spored/internal/event/sync"
	nodes "spored/internal/nodepool"
	reg_el "spored/internal/registry/element"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type registryLifecycleEndpoint struct {
	*base.Base
	messages chan string
}

func (e *registryLifecycleEndpoint) OnMessage(ctx message_event.IMessageContext) {
	e.messages <- ctx.Message()
}

func (e *registryLifecycleEndpoint) OnSynchronizedRequest(ctx sync_event.ISynchronizedRequestContext) {
	req, ok := cparser.ParseAndValidate(ctx.Message())
	if !ok || req.Command != "dev.sporeos.shell.confirm" {
		return
	}
	raw := fmt.Sprintf("~%s:%s ok yes", req.Handle, req.Command)
	ctx.Respond(message_event.Response(raw, "dev.sporeos.shell", req.Handle, ctx.Origin()))
}

func TestLifecycleSuffixUpdatesRealRegistry(t *testing.T) {
	for _, command := range []string{"uninstall", "reinstall"} {
		t.Run(command, func(t *testing.T) {
			event.Init()
			pool := nodes.New()
			pool.Open()
			t.Cleanup(pool.Close)
			r := New()
			r.Open()
			t.Cleanup(r.Close)
			endpoint := &registryLifecycleEndpoint{Base: base.New(), messages: make(chan string, 1)}
			event.Subscribe(endpoint)
			t.Cleanup(func() { event.Unsubscribe(endpoint) })

			dir := t.TempDir()
			path := filepath.Join(dir, "node.yaml")
			const id = "dev.sporeos.foundry.alias"
			if err := os.WriteFile(path, []byte("id: "+id+"\ntrust: developer\nnamespace: hyphae\napp: node\n"), 0600); err != nil {
				t.Fatal(err)
			}
			el := &reg_el.Element{ID: id, Manifest: path, Checksum: "developer", Binary: filepath.Join(dir, "node")}
			r.path = filepath.Join(dir, "registry.yaml")
			r.Elements = []*reg_el.Element{el}
			if !r.Save() {
				t.Fatal("could not save initial registry")
			}
			event.Dispatch().RegistryAnnouncement.Notify(registry_event.New(el))

			fullCommand := "dev.sporeos.SPORE.node." + command
			pool.OnRequest(message_event.Request(fullCommand+" node=alias ~h", "caller", fullCommand, "dev.sporeos.SPORE", "h"))
			select {
			case response := <-endpoint.messages:
				if !strings.HasSuffix(response, " ok") {
					t.Fatalf("unexpected response: %q", response)
				}
			case <-time.After(time.Second):
				t.Fatal("lifecycle request did not respond")
			}

			data, err := os.ReadFile(r.path)
			if err != nil {
				t.Fatal(err)
			}
			var saved struct {
				Elements []*reg_el.Element `yaml:"nodes"`
			}
			if err := yaml.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			if command == "uninstall" && len(saved.Elements) != 0 {
				t.Fatalf("uninstall left registry entries: %#v", saved.Elements)
			}
			if command == "reinstall" {
				if len(saved.Elements) != 1 || saved.Elements[0].ID != id || saved.Elements[0].Manifest != path {
					t.Fatalf("unexpected reinstalled registry entry: %#v", saved.Elements)
				}
				event.Dispatch().RegistryWithdrawal.Notify(registry_event.New(saved.Elements[0]))
			}
		})
	}
}

func TestUninstallationRestoresRegistryWhenSaveFails(t *testing.T) {
	event.Init()
	r := New()
	r.path = filepath.Join(t.TempDir(), "missing", "registry.yaml")
	first := &reg_el.Element{ID: "com.example.first"}
	second := &reg_el.Element{ID: "com.example.second"}
	r.Elements = []*reg_el.Element{first, second}

	r.OnNodeUninstallation(first.ID)

	if len(r.Elements) != 2 || r.Elements[0] != first || r.Elements[1] != second {
		t.Fatalf("registry changed after failed save: %#v", r.Elements)
	}
}
