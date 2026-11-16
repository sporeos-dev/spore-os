// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package error_test

import (
	"spored/internal/cparser"
	"spored/internal/iface"
	sporeerror "spored/internal/utilities/error"
	"testing"
)

type captureMessage struct {
	capture string
}

func (m captureMessage) Capability() string                  { return "missing" }
func (m captureMessage) Cast() string                        { return "caller" }
func (m captureMessage) Actor() string                       { return "caller" }
func (m captureMessage) Origin() string                      { return "caller" }
func (m captureMessage) Capture() string                     { return m.capture }
func (m captureMessage) Arg(string) (string, bool)           { return "", false }
func (m captureMessage) ArgIf(_ string, value string) string { return value }
func (m captureMessage) Flag(string) bool                    { return false }
func (m captureMessage) Handle() string                      { return "h1" }
func (m captureMessage) Wire() string                        { return "" }
func (m captureMessage) Witness() string                     { return "" }

func TestErrorCapture(t *testing.T) {
	for _, tt := range []struct {
		name    string
		message iface.Message
		want    string
	}{
		{name: "request", message: captureMessage{capture: "n/a"}, want: "dev.sporeos.SPORE"},
		{name: "response", message: captureMessage{capture: "responder"}, want: "responder"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := sporeerror.New(sporeerror.Generic, sporeerror.Bus, "failed").WithMessage(tt.message)
			if got := err.Capture(); got != tt.want {
				t.Fatalf("Capture() = %q, want %q", got, tt.want)
			}
			parsed, ok := cparser.Parse(err.Wire())
			if !ok {
				t.Fatalf("invalid wire response: %q", err.Wire())
			}
			if got := parsed.Args["capture"]; got != tt.want {
				t.Errorf("wire capture = %q, want %q", got, tt.want)
			}
			if parsed.Handle != "h1" || parsed.Command != "missing" {
				t.Errorf("wire response lost correlation: %q", err.Wire())
			}
		})
	}
}

func TestErrorWithoutMessageRemainsInternal(t *testing.T) {
	err := sporeerror.New(sporeerror.Generic, sporeerror.Bus, "failed")
	if err.Capture() != "n/a" || err.Wire() != "n/a" {
		t.Fatalf("error without message: capture=%q wire=%q", err.Capture(), err.Wire())
	}
}
