// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"testing"
)

// =============================================================================
// SPEC §2.4: The SPORE. Namespace
// "SPORE." is protocol-reserved. The hub exposes subjects for introspection
// and lifecycle management. SPORE messages are administrative.
// =============================================================================

func TestSpore_ParsesCommand(t *testing.T) {
	msg, err := Parse("SPORE.node.list ~meta", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Command() != "SPORE.node.list" {
		t.Errorf("expected SPORE.node.list, got %q", msg.Command())
	}
}

func TestSpore_ParsesHandle(t *testing.T) {
	msg, err := Parse("SPORE.node.list ~meta", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Handle() != "meta" {
		t.Errorf("expected handle meta, got %q", msg.Handle())
	}
}

func TestSpore_ParsesArguments(t *testing.T) {
	// SPEC §10: SPORE.node.install path=/path/to/manifest.yaml
	msg, err := Parse("SPORE.node.install ~h1 path=/path/to/manifest.yaml", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	spore := msg.(*Spore)
	val, getErr := spore.GetArgument("path")
	if getErr != nil {
		t.Fatalf("expected path argument, got error: %v", getErr)
	}
	if val != "/path/to/manifest.yaml" {
		t.Errorf("expected /path/to/manifest.yaml, got %q", val)
	}
}

func TestSpore_ParsesFlags(t *testing.T) {
	msg, err := Parse("SPORE.help ~h1 verbose", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	spore := msg.(*Spore)
	if !spore.HasFlag("verbose") {
		t.Error("expected verbose flag to be present")
	}
}

func TestSpore_CastIsCallerID(t *testing.T) {
	msg, err := Parse("SPORE.node.list ~meta", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Cast() != "com.example.agent" {
		t.Errorf("expected cast com.example.agent, got %q", msg.Cast())
	}
}

func TestSpore_DestinationIsSporeHub(t *testing.T) {
	// SPEC: SPORE messages are always destined for the hub
	msg, err := Parse("SPORE.node.list ~meta", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Destination() != "dev.sporeos.SPORE" {
		t.Errorf("expected destination dev.sporeos.SPORE, got %q", msg.Destination())
	}
}

func TestSpore_RejectsMessageWithNoHandle(t *testing.T) {
	// SPEC §6.2: every call must include a handle; the hub rejects calls without one
	_, err := Parse("SPORE.node.list", "com.example.agent")
	if err == nil {
		t.Fatal("expected error for missing handle, got nil")
	}
}

func TestSpore_TypeFlags(t *testing.T) {
	msg, err := Parse("SPORE.node.list ~meta", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsSpore() {
		t.Error("expected IsSpore true")
	}
	if msg.IsCast() {
		t.Error("SPORE message should not be IsCast")
	}
	if msg.IsCapture() {
		t.Error("SPORE message should not be IsCapture")
	}
	if msg.IsError() {
		t.Error("SPORE message should not be IsError")
	}
}

func TestSpore_HandleIsOptional(t *testing.T) {
	// SPEC §6.2: every call must include a handle; the hub rejects calls without one.
	// SPORE messages are calls and follow the same rule.
	_, err := Parse("SPORE.help", "com.example.agent")
	if err == nil {
		t.Fatal("expected error for missing handle, got nil")
	}
}

func TestSpore_GetArgumentReturnsErrorIfMissing(t *testing.T) {
	msg, _ := Parse("SPORE.node.list ~meta", "com.example.agent")
	spore := msg.(*Spore)
	_, err := spore.GetArgument("nonexistent")
	if err == nil {
		t.Error("expected error for missing argument")
	}
}

func TestSpore_GetArgumentIfReturnsDefault(t *testing.T) {
	msg, _ := Parse("SPORE.node.list ~meta", "com.example.agent")
	spore := msg.(*Spore)
	val := spore.GetArgumentIf("node", "n/a")
	if val != "n/a" {
		t.Errorf("expected default n/a, got %q", val)
	}
}

func TestSpore_ArgsAndFlagsWorkWithHandleInAnyPosition(t *testing.T) {
	// Handle may appear before or after args/flags — parser must not stop early
	cases := []struct {
		name string
		raw  string
	}{
		{"handle last", "SPORE.node.install path=/tmp/a.yaml ~h1"},
		{"handle first", "SPORE.node.install ~h1 path=/tmp/a.yaml"},
		{"handle middle", "SPORE.node.install path=/tmp/a.yaml ~h1 verbose"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := Parse(tc.raw, "com.example.agent")
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}
			s := msg.(*Spore)
			if s.Handle() != "h1" {
				t.Errorf("expected handle h1, got %q", s.Handle())
			}
			val, getErr := s.GetArgument("path")
			if getErr != nil {
				t.Errorf("expected path argument, got error: %v", getErr)
			}
			if val != "/tmp/a.yaml" {
				t.Errorf("expected /tmp/a.yaml, got %q", val)
			}
		})
	}
}

func TestSpore_SetDestinationIsNoop(t *testing.T) {
	// SPORE messages always go to the hub; SetDestination should not change it
	msg, _ := Parse("SPORE.node.list ~meta", "com.example.agent")
	msg.SetDestination("com.example.evil")
	if msg.Destination() != "dev.sporeos.SPORE" {
		t.Errorf("SPORE destination should not change, got %q", msg.Destination())
	}
}

// =============================================================================
// SPEC §10: Hub Manifest API — All known SPORE commands
// =============================================================================

func TestSpore_AllHubCommandsParse(t *testing.T) {
	// Every SPORE.* command from the hub manifest should parse as a Spore message
	commands := []string{
		"SPORE.help ~h1",
		"SPORE.node.install ~h2 path=/tmp/manifest.yaml",
		"SPORE.node.uninstall ~h3 node=com.example.clock",
		"SPORE.node.spawn ~h4 node=com.example.clock",
		"SPORE.node.kill ~h5 node=com.example.clock",
		"SPORE.node.list ~h6",
		"SPORE.node.help ~h7 node=com.example.clock",
		"SPORE.command.list ~h8",
		"SPORE.command.help ~h9 command=clock.get_time",
		"SPORE.error.list ~h10",
		"SPORE.error.help ~h11 code=RouteNotFound",
	}
	for _, raw := range commands {
		msg, err := Parse(raw, "com.example.agent")
		if err != nil {
			t.Errorf("command %q failed to parse: %v", raw, err)
			continue
		}
		if !msg.IsSpore() {
			t.Errorf("command %q should be Spore type", raw)
		}
	}
}
