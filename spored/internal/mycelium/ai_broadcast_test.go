package mycelium

import (
	"spored/internal/event"
	"testing"
)

func TestExtractTopic(t *testing.T) {
	event.Init()

	tests := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{"topic and payload", "some.topic value=one", "some.topic", true},
		{"topic only", "some.topic", "some.topic", true},
		{"leading whitespace", " \tsome.topic\nvalue=one", "some.topic", true},
		{"unicode whitespace", "\u2003some.topic\u2003value=one", "some.topic", true},
		{"empty input", "", "", false},
		{"whitespace only", " \t\n", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := extractTopic(tt.raw)
			if got != tt.want || ok != tt.ok {
				t.Errorf("extractTopic(%q) = %q, %t; want %q, %t", tt.raw, got, ok, tt.want, tt.ok)
			}
		})
	}
}
