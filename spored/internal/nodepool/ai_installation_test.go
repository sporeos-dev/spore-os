package nodes

import (
	"os"
	"path/filepath"
	"spored/internal/base"
	"spored/internal/event"
	node_event "spored/internal/event/node"
	"testing"
)

type installationHandler struct {
	*base.Base
	acknowledge bool
	calls       int
}

func (h *installationHandler) OnNodeInstallation(ctx node_event.INodeInstallationContext) {
	h.calls++
	if h.acknowledge {
		ctx.SetInstalled()
	}
}

func TestNodeInstallValidatesAndRequiresRegistryAcknowledgment(t *testing.T) {
	tests := []struct {
		name        string
		extra       string
		acknowledge bool
		want        bool
		wantCalls   int
	}{
		{"registered", "", true, true, 1},
		{"registry failure", "", false, false, 1},
		{"reserved name", "api:\n  - name: call\n    inputs:\n      - name: cast\n", true, false, 0},
		{"invalid namespace", "namespace: any\n", true, false, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event.Init()
			handler := &installationHandler{Base: base.New(), acknowledge: tt.acknowledge}
			event.Subscribe(handler)
			t.Cleanup(func() { event.Unsubscribe(handler) })

			path := filepath.Join(t.TempDir(), "node.yaml")
			content := "id: example.node\ntrust: standard\napp: binary\n" + tt.extra
			if err := os.WriteFile(path, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}

			if got := New().nodeInstall(path); got != tt.want || handler.calls != tt.wantCalls {
				t.Fatalf("nodeInstall() = %t, handler calls = %d; want %t, %d", got, handler.calls, tt.want, tt.wantCalls)
			}
		})
	}
}
