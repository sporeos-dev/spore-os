// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"strings"
	"testing"
)

// =============================================================================
// SPEC §6.1: Cast — "subject [key=value ...] [flag ...] ~handle"
// The caller sends a subject with optional arguments and a handle. Before
// forwarding to the receiver, the hub injects cast= identifying the caller.
// =============================================================================

func TestCast_ParsesSubjectAndHandle(t *testing.T) {
	msg, err := Parse("clock.get_time ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Command() != "clock.get_time" {
		t.Errorf("expected command clock.get_time, got %q", msg.Command())
	}
	if msg.Handle() != "h1" {
		t.Errorf("expected handle h1, got %q", msg.Handle())
	}
}

func TestCast_InjectsCastField(t *testing.T) {
	// SPEC §6.1: "Before forwarding to the receiver, the hub injects cast=
	// identifying the originating caller."
	msg, err := Parse("clock.get_time timezone=UTC ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wire := msg.ToString()
	if !strings.Contains(wire, "cast=com.example.agent") {
		t.Errorf("expected cast= injection in wire output, got %q", wire)
	}
}

func TestCast_SourceIsCaller(t *testing.T) {
	msg, err := Parse("clock.get_time ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Source() != "com.example.agent" {
		t.Errorf("expected source com.example.agent, got %q", msg.Source())
	}
	if msg.Cast() != "com.example.agent" {
		t.Errorf("expected cast com.example.agent, got %q", msg.Cast())
	}
}

func TestCast_RejectsWithoutHandle(t *testing.T) {
	// SPEC §6.2: "Every call must include a handle. The hub rejects calls without one."
	_, err := Parse("clock.get_time timezone=UTC", "com.example.agent")
	if err == nil {
		t.Error("expected error when handle is missing from cast")
	}
}

func TestCast_PreservesArguments(t *testing.T) {
	// SPEC §6.3: "key=value" argument syntax
	msg, err := Parse("clock.get_time timezone=UTC ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wire := msg.ToString()
	if !strings.Contains(wire, "timezone=UTC") {
		t.Errorf("expected timezone=UTC in wire output, got %q", wire)
	}
}

func TestCast_HandlesFlags(t *testing.T) {
	// SPEC §6.3: Flags — bare tokens
	msg, err := Parse("filesystem.read path=/tmp recursive ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cast := msg.(*Cast)
	if !cast.HasFlag("recursive") {
		t.Error("expected recursive flag to be present")
	}
}

func TestCast_JsonFlagIsRecognized(t *testing.T) {
	// SPEC §6.6: json flag — "Request JSON-formatted output"
	msg, err := Parse("clock.get_time ~h1 json", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cast := msg.(*Cast)
	if !cast.HasFlag("json") {
		t.Error("expected json flag to be present")
	}
}

func TestCast_ShortFormSubject(t *testing.T) {
	// SPEC §5.1: Short form — "clock.get_time"
	msg, err := Parse("clock.get_time ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Command() != "clock.get_time" {
		t.Errorf("expected short form command, got %q", msg.Command())
	}
}

func TestCast_FullFormSubject(t *testing.T) {
	// SPEC §5.1: Full form — "com.example.clock.get_time"
	msg, err := Parse("com.example.clock.get_time ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Command() != "com.example.clock.get_time" {
		t.Errorf("expected full form command, got %q", msg.Command())
	}
}

func TestCast_CaptureFieldIsEmpty(t *testing.T) {
	// A cast has no capture (it hasn't been responded to yet)
	msg, err := Parse("clock.get_time ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Capture() != "" {
		t.Errorf("expected empty capture on cast, got %q", msg.Capture())
	}
}

func TestCast_MessageIdDefaultsToNegativeOne(t *testing.T) {
	msg, err := Parse("clock.get_time ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.MessageId() != -1 {
		t.Errorf("expected default message ID -1, got %d", msg.MessageId())
	}
}

func TestCast_ToJsonStringContainsCommandAndHandle(t *testing.T) {
	msg, err := Parse("clock.get_time timezone=UTC ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	js := msg.ToJsonString()
	if !strings.Contains(js, `"command":"clock.get_time"`) {
		t.Errorf("JSON missing command, got %q", js)
	}
	if !strings.Contains(js, `"handle":"h1"`) {
		t.Errorf("JSON missing handle, got %q", js)
	}
}
