package message

import (
	"testing"
)

// =============================================================================
// SPEC §6: Wire Grammar — Message Dispatch
// The Parse dispatcher determines message type by prefix:
//   - "SPORE." → Spore message
//   - "~"      → Capture or Error
//   - anything else → Cast
// =============================================================================

func TestParse_DispatchesCastForPlainSubject(t *testing.T) {
	// SPEC §6.1: "subject [key=value ...] [flag ...] ~handle"
	msg, err := Parse("clock.get_time ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsCast() {
		t.Error("expected Cast message for plain subject")
	}
	if msg.IsCapture() || msg.IsSpore() || msg.IsError() {
		t.Error("wrong type flags set on Cast")
	}
}

func TestParse_DispatchesCaptureForTildePrefix(t *testing.T) {
	// SPEC §6.1: "~handle:subject [key=value ...] ok capture=node_id"
	msg, err := Parse("~h1:clock.get_time time=\"2026-03-13T14:32:00Z\"", "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsCapture() {
		t.Error("expected Capture message for ~handle:subject prefix")
	}
}

func TestParse_DispatchesErrorForTildePrefixWithErrorFlag(t *testing.T) {
	// SPEC §6.5: "~handle:subject error code=CODE what=\"description\""
	msg, err := Parse(`~h1:clock.get_time error code=Runtime what="Internal clock failure" node_error`, "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsError() {
		t.Error("expected Error message when error flag is present")
	}
}

func TestParse_DispatchesSporeForSporePrefix(t *testing.T) {
	// SPEC §2.4: "SPORE." is protocol-reserved
	msg, err := Parse("SPORE.node.list ~meta", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsSpore() {
		t.Error("expected Spore message for SPORE. prefix")
	}
}

func TestParse_RejectsEmptyMessage(t *testing.T) {
	_, err := Parse("", "com.example.agent")
	if err == nil {
		t.Error("expected error for empty message")
	}
}

func TestParse_RejectsWhitespaceOnly(t *testing.T) {
	_, err := Parse("   \t  ", "com.example.agent")
	if err == nil {
		t.Error("expected error for whitespace-only message")
	}
}

// =============================================================================
// SPEC §6.2: ~handle — ExtractHandle
// Even when parse fails, a handle may still be present for error feedback.
// =============================================================================

func TestExtractHandle_FromCastShape(t *testing.T) {
	// "subject ... ~handle"
	h := ExtractHandle("clock.get_time timezone=UTC ~h1")
	if h != "h1" {
		t.Errorf("expected h1, got %q", h)
	}
}

func TestExtractHandle_FromCaptureShape(t *testing.T) {
	// "~handle:subject ..."
	h := ExtractHandle("~h1:clock.get_time ok capture=com.example.clock")
	if h != "h1" {
		t.Errorf("expected h1, got %q", h)
	}
}

func TestExtractHandle_ReturnsEmptyIfNoHandle(t *testing.T) {
	h := ExtractHandle("clock.get_time timezone=UTC")
	if h != "" {
		t.Errorf("expected empty, got %q", h)
	}
}

func TestExtractHandle_FromMalformedMessage(t *testing.T) {
	// Even garbage with a ~token should extract
	h := ExtractHandle("some garbage ~myhandle more stuff")
	if h != "myhandle" {
		t.Errorf("expected myhandle, got %q", h)
	}
}

// =============================================================================
// SPEC §6.1: Cancelled Response
// "~handle:subject cancelled capture=node_id"
// "A cancelled response indicates a graceful non-result."
// =============================================================================

func TestParse_DispatchesCancelledForCancelledFlag(t *testing.T) {
	msg, err := Parse("~h1:dialog.file_picker cancelled", "com.example.dialog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsCancelled() {
		t.Error("expected Cancelled message when cancelled flag is present")
	}
	if msg.IsCapture() || msg.IsError() || msg.IsCast() || msg.IsSpore() {
		t.Error("wrong type flags set on Cancelled")
	}
}

func TestCancelled_ParsesHandleAndCommand(t *testing.T) {
	msg, err := Parse("~h1:dialog.file_picker cancelled", "com.example.dialog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Handle() != "h1" {
		t.Errorf("expected handle h1, got %q", msg.Handle())
	}
	if msg.Command() != "dialog.file_picker" {
		t.Errorf("expected command dialog.file_picker, got %q", msg.Command())
	}
}

func TestCancelled_InjectsCaptureFromSender(t *testing.T) {
	// SPEC: "The hub injects capture= as it does for all responses."
	msg, err := Parse("~h1:dialog.file_picker cancelled", "com.example.dialog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Capture() != "com.example.dialog" {
		t.Errorf("expected capture=com.example.dialog, got %q", msg.Capture())
	}
}

func TestCancelled_ToStringIncludesCaptureInjection(t *testing.T) {
	msg, err := Parse("~h1:dialog.file_picker cancelled", "com.example.dialog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw := msg.ToString()
	if !containsCancelledFlag(raw) {
		t.Error("ToString should contain cancelled flag")
	}
	if !contains(raw, "capture=com.example.dialog") {
		t.Errorf("ToString should contain capture=, got: %s", raw)
	}
}

func TestCancelled_WithExistingCaptureUsesIt(t *testing.T) {
	// If capture= is already present in the raw string, Parse should still work
	msg, err := Parse("~h1:dialog.file_picker cancelled capture=com.example.other", "com.example.dialog")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The explicit capture= in the message takes precedence
	if msg.Capture() != "com.example.other" {
		t.Errorf("expected capture=com.example.other, got %q", msg.Capture())
	}
}

func TestNewCancelled_BuildsHubGeneratedCancelled(t *testing.T) {
	// Parse an original cast message first
	original, err := Parse("dialog.file_picker ~h1", "com.example.agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cancelled := NewCancelled(original)
	if cancelled.Handle() != "h1" {
		t.Errorf("expected handle h1, got %q", cancelled.Handle())
	}
	if cancelled.Command() != "dialog.file_picker" {
		t.Errorf("expected command dialog.file_picker, got %q", cancelled.Command())
	}
	if cancelled.Capture() != "SPORE.hub" {
		t.Errorf("expected capture=SPORE.hub, got %q", cancelled.Capture())
	}
	if !cancelled.IsCancelled() {
		t.Error("should be cancelled")
	}
	raw := cancelled.ToString()
	if !contains(raw, "cancelled") {
		t.Error("ToString should contain cancelled")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
