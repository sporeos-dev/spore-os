package manifest

import (
	"spored/internal/utilities/status"
	"strings"
	"testing"
)

type validationNotifier struct {
	announcements int
}

func (n *validationNotifier) Announce(*Manifest) { n.announcements++ }
func (n *validationNotifier) Withdraw(*Manifest) {}

func TestLoadContentValidatesBeforeAnnouncement(t *testing.T) {
	tests := []struct {
		name    string
		content string
		valid   bool
	}{
		{"default namespace", "id: example.node\n", true},
		{"spore namespace", "id: example.node\nnamespace: spore\n", true},
		{"removed namespace", "id: example.node\nnamespace: any\n", false},
		{"reserved node id", "id: example:node\n", false},
		{"reserved input", "id: example.node\napi:\n  - name: call\n    inputs:\n      - name: spore.custom\n", false},
		{"reserved topic output", "id: example.node\ntopics:\n  - name: topic\n    outputs:\n      - name: capture\n", false},
		{"allowed names", "id: example.node\napi:\n  - name: call\n    inputs:\n      - name: spore_custom\n      - name: module\n      - name: timeout\n", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			notifier := &validationNotifier{}
			m := Empty("", "")
			m.Notifier = notifier
			err := m.LoadContent(tt.content)
			if (err == nil) != tt.valid {
				t.Fatalf("LoadContent() error = %v, valid = %t", err, tt.valid)
			}
			if tt.valid {
				if notifier.announcements != 1 {
					t.Fatalf("announcements = %d, want 1", notifier.announcements)
				}
			} else if m.Status.Get() != status.Malformed || notifier.announcements != 0 {
				t.Fatalf("status = %s, announcements = %d", m.Status.Get(), notifier.announcements)
			}
			if tt.name == "default namespace" && !strings.EqualFold(string(m.Namespace), "hyphae") {
				t.Fatalf("namespace = %q, want hyphae", m.Namespace)
			}
		})
	}
}
