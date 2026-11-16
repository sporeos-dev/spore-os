package registry

import (
	"os"
	"path/filepath"
	"spored/internal/event"
	node_event "spored/internal/event/node"
	"spored/internal/manifest"
	"testing"
)

func TestInstallationAcknowledgesOnlyAfterRegistrySave(t *testing.T) {
	for _, success := range []bool{true, false} {
		t.Run(map[bool]string{true: "saved", false: "save failed"}[success], func(t *testing.T) {
			event.Init()
			r := New()
			dir := t.TempDir()
			r.path = filepath.Join(dir, "registry.yaml")
			if !success {
				r.path = filepath.Join(dir, "missing", "registry.yaml")
			}
			m := manifest.Empty(filepath.Join(dir, "node.yaml"), "")
			m.ID = "example.node"
			m.Trust = manifest.Developer
			ctx := node_event.New(m)

			r.OnNodeInstallation(ctx)

			if ctx.Installed() != success {
				t.Fatalf("Installed() = %t, want %t", ctx.Installed(), success)
			}
			if success {
				if len(r.Elements) != 1 {
					t.Fatalf("registry elements = %d, want 1", len(r.Elements))
				}
				if _, err := os.Stat(r.path); err != nil {
					t.Fatalf("registry was not saved: %v", err)
				}
			} else if len(r.Elements) != 0 {
				t.Fatalf("failed installation left %d registry elements", len(r.Elements))
			}
		})
	}
}
