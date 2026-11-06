// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package error_test

import (
	"spored/internal/cparser"
	"spored/internal/iface"
	"spored/internal/message"
	sporeerror "spored/internal/utilities/error"
	"testing"
)

func TestErrorCapture(t *testing.T) {
	request, ok := message.Request("missing ~h1", "caller")
	if !ok {
		t.Fatal("invalid request")
	}
	response, ok := message.Response(`~h1:missing error code=Generic.Node what="failed"`, "responder")
	if !ok {
		t.Fatal("invalid response")
	}

	for _, tt := range []struct {
		name    string
		message iface.Message
		want    string
	}{
		{name: "request", message: request, want: "dev.sporeos.SPORE"},
		{name: "response", message: response, want: "responder"},
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
