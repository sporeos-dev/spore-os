package message

import (
	"strings"
	"testing"
)

// =============================================================================
// SPEC §6.1: Capture — "~handle:subject [key=value ...] ok capture=node_id"
// Every call receives exactly one response. The ~handle:subject prefix binds
// the response to its originating call. The hub injects ok and capture=.
// =============================================================================

func TestCapture_ParsesHandleAndCommand(t *testing.T) {
	msg, err := Parse("~h1:clock.get_time time=\"2026-03-13T14:32:00Z\"", "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Handle() != "h1" {
		t.Errorf("expected handle h1, got %q", msg.Handle())
	}
	if msg.Command() != "clock.get_time" {
		t.Errorf("expected command clock.get_time, got %q", msg.Command())
	}
}

func TestCapture_InjectsOkFlag(t *testing.T) {
	// SPEC §6.1: "The hub injects ok and capture= on every success response"
	msg, err := Parse("~h1:clock.get_time time=\"2026-03-13T14:32:00Z\"", "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wire := msg.ToString()
	if !strings.Contains(wire, " ok ") && !strings.HasSuffix(wire, " ok") {
		// ok should be present somewhere in the wire output
		if !strings.Contains(wire, "ok") {
			t.Errorf("expected ok in wire output, got %q", wire)
		}
	}
}

func TestCapture_InjectsCaptureField(t *testing.T) {
	// SPEC §6.1: "The hub injects ... capture= on every success response"
	msg, err := Parse("~h1:clock.get_time time=\"2026-03-13T14:32:00Z\"", "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wire := msg.ToString()
	if !strings.Contains(wire, "capture=com.example.clock") {
		t.Errorf("expected capture= injection, got %q", wire)
	}
}

func TestCapture_ResponseIsSelfDescribing(t *testing.T) {
	// SPEC §6.2: "~handle:subject — the :subject portion echoes what was called,
	// making every response self-describing"
	msg, err := Parse("~h1:clock.get_time time=\"2026-03-13T14:32:00Z\"", "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wire := msg.ToString()
	if !strings.HasPrefix(wire, "~h1:clock.get_time") {
		t.Errorf("expected wire to start with ~h1:clock.get_time, got %q", wire)
	}
}

func TestCapture_SourceIsResponder(t *testing.T) {
	msg, err := Parse("~h1:clock.get_time time=\"2026-03-13\"", "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Source() != "com.example.clock" {
		t.Errorf("expected source com.example.clock, got %q", msg.Source())
	}
}

func TestCapture_CaptureFieldMatchesResponder(t *testing.T) {
	msg, err := Parse("~h1:clock.get_time time=\"2026-03-13\"", "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Capture() != "com.example.clock" {
		t.Errorf("expected capture com.example.clock, got %q", msg.Capture())
	}
}

func TestCapture_IsNotCast(t *testing.T) {
	msg, err := Parse("~h1:clock.get_time result=ok", "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.IsCast() {
		t.Error("capture should not report IsCast")
	}
	if msg.IsSpore() {
		t.Error("capture should not report IsSpore")
	}
	if msg.IsError() {
		t.Error("success capture should not report IsError")
	}
}

func TestCapture_PreservesResponseData(t *testing.T) {
	msg, err := Parse("~r1:filesystem.read content=\"Q1 planning notes...\"", "com.example.filesystem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wire := msg.ToString()
	if !strings.Contains(wire, "content=\"Q1 planning notes...\"") {
		t.Errorf("expected response data preserved, got %q", wire)
	}
}

func TestCapture_ToJsonStringContainsFields(t *testing.T) {
	msg, err := Parse("~h1:clock.get_time time=now", "com.example.clock")
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
