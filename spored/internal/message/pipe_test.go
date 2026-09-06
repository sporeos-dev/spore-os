package message

import (
	"testing"
)

func TestIsPipe(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{"dialog.file_picker ~h1 | SPORE.node.install", true},
		{"source.open ~h2 | transform.normalize | destination.write", true},
		{"notify.send msg=\"hello | world\" ~h3", false},
		{"array.get item=['a|b', 'c|d'] ~h4", false},
		{"simple.call arg=val ~h5", false},
	}

	for _, tt := range tests {
		got := IsPipe(tt.raw)
		if got != tt.want {
			t.Errorf("IsPipe(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestPipeCapability_EmptyAfterFinalPart(t *testing.T) {
	msg, ok := Pipe("one | two", "test_node")
	if !ok {
		t.Fatalf("Pipe failed")
	}

	if got := msg.Capability(); got != "one" {
		t.Errorf("first capability: got %q, want %q", got, "one")
	}
	if got := msg.Capability(); got != "two" {
		t.Errorf("second capability: got %q, want %q", got, "two")
	}
	if got := msg.Capability(); got != "" {
		t.Errorf("exhausted capability: got %q, want empty", got)
	}
}
