// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package cparser

import (
	"spored/internal/event"
	"testing"
)

func TestParsedArgumentsPreserveDecodedValues(t *testing.T) {
	event.Init()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"double quoted", `example.run path="a \"quote\" \\path\nline\rreturn\ttab" ~h`, "a \"quote\" \\path\nline\rreturn\ttab"},
		{"unknown escape", `example.run path="a\q" ~h`, `a\q`},
		{"single quoted literal", `example.run path='a\n"quoted"\path' ~h`, `a\n"quoted"\path`},
		{"literal enclosing quotes", `example.run path='"quoted"' ~h`, `"quoted"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, ok := ParseAndValidate(tt.raw)
			if !ok {
				t.Fatalf("failed to parse %q", tt.raw)
			}
			if got := msg.Args["path"]; got != tt.want {
				t.Errorf("path = %q, want %q", got, tt.want)
			}
		})
	}
}
