// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package mycelium

import (
	"fmt"
	"os"
	"path/filepath"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	"spored/internal/manifest"
	nodes "spored/internal/nodepool"
	"strings"
	"testing"
)

func TestPipeReceivesNodePoolErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invalid.yaml")
	if err := os.WriteFile(path, []byte("namespace: invalid\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		pipe string
		want string
	}{
		{"missing install path", "source.read ~h | install", "Missing.NodePool"},
		{"installation failure", fmt.Sprintf("source.read ~h | install path='%s'", path), "Generic.NodePool"},
		{"missing reinstall node", "source.read ~h | reinstall", "Missing.NodePool"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, endpoint := setupPipeTest(t, func(ctx message_event.IRequestContext) string {
				return fmt.Sprintf("~%s:source.read ok", ctx.Handle())
			})
			pool := nodes.New()
			pool.Open()
			t.Cleanup(pool.Close)
			pathInput := []manifest.Input{{Name: "path"}}
			nodeInput := []manifest.Input{{Name: "node"}}
			m.commands.OnManifestAnnouncement(manifest_event.New(&manifest.Manifest{
				ID: "dev.sporeos.SPORE",
				Api: []manifest.Command{
					{Name: "node.install", Inputs: &pathInput},
					{Name: "node.reinstall", Inputs: &nodeInput},
				},
			}))

			m.OnMessage(message_event.Raw(tt.pipe, "caller"))
			response := pipeResult(t, endpoint)
			if response.Handle() != "h" || !strings.Contains(response.Message(), tt.want) {
				t.Fatalf("response = %q, want handle h and %s", response.Message(), tt.want)
			}
			select {
			case extra := <-endpoint.responses:
				t.Fatalf("unexpected extra response: %q", extra.Message())
			default:
			}
		})
	}
}
