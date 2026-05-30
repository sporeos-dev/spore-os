// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"strings"
	"testing"
)

// =============================================================================
// SPEC §6.5: Errors
// Error responses follow the same grammar as success responses. The presence
// of "error" or "custom_error" marks the response as an error; code= carries
// a standard or custom identifier; what= carries a human-readable description.
// =============================================================================

// --- NodeError (wire-parsed errors from nodes) ---

func TestNodeError_ParsesStandardError(t *testing.T) {
	// SPEC §6.5: "~handle:subject error code=CODE what=\"description\" node_error"
	raw := `~h1:time.get_time error code=Runtime what="Internal clock failure" node_error`
	msg, err := Parse(raw, "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsError() {
		t.Error("expected IsError")
	}
	if msg.IsCustomError() {
		t.Error("standard error should not be IsCustomError")
	}
	ne := msg.(*NodeError)
	if ne.Code() != ErrorCodeRuntime {
		t.Errorf("expected code Runtime, got %q", ne.Code())
	}
	if ne.What() != "Internal clock failure" {
		t.Errorf("expected what string, got %q", ne.What())
	}
	if ne.Handle() != "h1" {
		t.Errorf("expected handle h1, got %q", ne.Handle())
	}
	if ne.Command() != "time.get_time" {
		t.Errorf("expected command time.get_time, got %q", ne.Command())
	}
}

func TestNodeError_ParsesCustomError(t *testing.T) {
	// SPEC §6.5: "custom_error — used by a node when the error corresponds
	// to a named entry in its manifest errors field"
	raw := `~h1:time.get_time custom_error code=clock.err.invalid_timezone what="Unknown timezone: Fakezone" node_error`
	msg, err := Parse(raw, "com.example.clock")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsError() {
		t.Error("expected IsError")
	}
	if !msg.IsCustomError() {
		t.Error("expected IsCustomError for custom_error flag")
	}
	ne := msg.(*NodeError)
	if ne.Code() != "clock.err.invalid_timezone" {
		t.Errorf("expected custom code, got %q", ne.Code())
	}
}

func TestNodeError_BlocksSporeFailureFromNodes(t *testing.T) {
	// SPEC §6.5: "SporeFailure — Hub-only — the hub blocks this code if a node emits it."
	raw := `~h1:foo.bar error code=SporeFailure what="sneaky" node_error`
	msg, err := Parse(raw, "com.example.evil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ne := msg.(*NodeError)
	if ne.Code() == ErrorCodeSporeFailure {
		t.Error("SporeFailure must be blocked when emitted by a node")
	}
	if ne.Code() != ErrorCodeProtocolFailure {
		t.Errorf("expected SporeFailure to be replaced with ProtocolFailure, got %q", ne.Code())
	}
}

func TestNodeError_DefaultOriginFlagIsNodeError(t *testing.T) {
	// If no origin flag is supplied, default to node_error
	raw := `~h1:foo.bar error code=Runtime what="oops"`
	msg, err := Parse(raw, "com.example.node")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ne := msg.(*NodeError)
	if ne.originFlag != "node_error" {
		t.Errorf("expected default origin node_error, got %q", ne.originFlag)
	}
}

// --- Hub-generated errors (CastError, CaptureError, SporeError) ---

func TestCastError_WireFormat(t *testing.T) {
	// SPEC §6.5: "~h1:subject error cast_error code=CODE what=\"...\" capture=SPORE.hub"
	original, _ := Parse("clock.get_time ~h1", "com.example.agent")
	errMsg := NewError(original, ErrorCodeRouteNotFound, "No node connected for this subject")

	wire := errMsg.ToString()
	if !strings.Contains(wire, "~h1:clock.get_time") {
		t.Errorf("expected ~handle:subject, got %q", wire)
	}
	if !strings.Contains(wire, "error") {
		t.Errorf("expected error flag, got %q", wire)
	}
	if !strings.Contains(wire, "cast_error") {
		t.Errorf("expected cast_error origin, got %q", wire)
	}
	if !strings.Contains(wire, "code=RouteNotFound") {
		t.Errorf("expected code=RouteNotFound, got %q", wire)
	}
	if !strings.Contains(wire, "capture=SPORE.hub") {
		t.Errorf("expected capture=SPORE.hub, got %q", wire)
	}
}

func TestCaptureError_WireFormat(t *testing.T) {
	// Error when routing a capture back
	original, _ := Parse("~h1:clock.get_time time=now", "com.example.clock")
	errMsg := NewError(original, ErrorCodeConnectionFailure, "caster gone")

	wire := errMsg.ToString()
	if !strings.Contains(wire, "capture_error") {
		t.Errorf("expected capture_error origin, got %q", wire)
	}
	if !strings.Contains(wire, "capture=SPORE.hub") {
		t.Errorf("expected capture=SPORE.hub, got %q", wire)
	}
}

func TestSporeError_WireFormat(t *testing.T) {
	// Error when a SPORE.* command fails
	original, _ := Parse("SPORE.node.list ~meta", "com.example.agent")
	errMsg := NewError(original, ErrorCodeSporeFailure, "internal hub error")

	wire := errMsg.ToString()
	if !strings.Contains(wire, "spore_error") {
		t.Errorf("expected spore_error origin, got %q", wire)
	}
	if !strings.Contains(wire, "code=SporeFailure") {
		t.Errorf("expected code=SporeFailure, got %q", wire)
	}
}

func TestNewError_PreservesOriginalMessage(t *testing.T) {
	original, _ := Parse("clock.get_time ~h1", "com.example.agent")
	errMsg := NewError(original, ErrorCodeRouteNotFound, "not found")

	if errMsg.Handle() != "h1" {
		t.Errorf("expected handle h1 from original, got %q", errMsg.Handle())
	}
	if errMsg.Command() != "clock.get_time" {
		t.Errorf("expected command from original, got %q", errMsg.Command())
	}
	if errMsg.Cast() != "com.example.agent" {
		t.Errorf("expected cast from original, got %q", errMsg.Cast())
	}
}

func TestNewError_MatchesOriginalType(t *testing.T) {
	// CastError for casts
	castMsg, _ := Parse("clock.get_time ~h1", "com.example.agent")
	castErr := NewError(castMsg, ErrorCodeGeneric, "err")
	if !castErr.IsCast() {
		t.Error("expected CastError for cast original")
	}

	// CaptureError for captures
	capMsg, _ := Parse("~h1:clock.get_time result=ok", "com.example.clock")
	capErr := NewError(capMsg, ErrorCodeGeneric, "err")
	if !capErr.IsCapture() {
		t.Error("expected CaptureError for capture original")
	}

	// SporeError for spore
	sporeMsg, _ := Parse("SPORE.node.list ~m1", "com.example.agent")
	sporeErr := NewError(sporeMsg, ErrorCodeGeneric, "err")
	if !sporeErr.IsSpore() {
		t.Error("expected SporeError for spore original")
	}
}

// --- Standard error codes coverage ---

func TestStandardErrors_ContainsAllSpecCodes(t *testing.T) {
	// SPEC §6.5: All standard error codes must be defined
	specCodes := []ErrorCode{
		// Internal / meta
		ErrorCodeUnknownFailure, ErrorCodeSporeFailure, ErrorCodeProtocolFailure, ErrorCodeConnectionFailure,
		// General
		ErrorCodeGeneric, ErrorCodeFatal, ErrorCodeTimeout, ErrorCodeBusy,
		ErrorCodeResourcesExhausted, ErrorCodeDeprecated, ErrorCodeRuntime, ErrorCodeLogic,
		// Route
		ErrorCodeRouteNotFound, ErrorCodeRouteNotConnected, ErrorCodeRouteNotAvailable,
		ErrorCodeRouteNotAllowed, ErrorCodeRouteNotImplemented,
		// Message
		ErrorCodeMessageNotValid, ErrorCodeMessageMalformed,
		// Arguments
		ErrorCodeArgumentMissing, ErrorCodeArgumentInvalidType, ErrorCodeArgumentConflict,
		ErrorCodeArgumentOutOfRange, ErrorCodeArgumentUnrecognized, ErrorCodeArgumentDuplicated,
		// Flags
		ErrorCodeFlagConflict, ErrorCodeFlagUnrecognized, ErrorCodeFlagDuplicated,
		// Handles
		ErrorCodeHandleMissing, ErrorCodeHandleInUse, ErrorCodeHandleExpired,
		// Reserved
		ErrorCodeReservedKeyword,
	}

	errors := StandardErrors()
	codeSet := make(map[ErrorCode]bool)
	for _, e := range errors {
		codeSet[e.Code] = true
	}

	for _, code := range specCodes {
		if !codeSet[code] {
			t.Errorf("standard error code %q is in SPEC but missing from StandardErrors()", code)
		}
	}
}

func TestAllErrorTypes_ReportIsError(t *testing.T) {
	original, _ := Parse("clock.get_time ~h1", "com.example.agent")
	castErr := NewError(original, ErrorCodeGeneric, "err")
	if !castErr.IsError() {
		t.Error("CastError should report IsError")
	}

	capOriginal, _ := Parse("~h1:clock.get_time result=ok", "com.example.clock")
	capErr := NewError(capOriginal, ErrorCodeGeneric, "err")
	if !capErr.IsError() {
		t.Error("CaptureError should report IsError")
	}

	sporeOriginal, _ := Parse("SPORE.node.list ~m1", "com.example.agent")
	sporeErr := NewError(sporeOriginal, ErrorCodeGeneric, "err")
	if !sporeErr.IsError() {
		t.Error("SporeError should report IsError")
	}

	// NodeError (wire-parsed)
	raw := `~h1:foo.bar error code=Runtime what="oops" node_error`
	nodeErr, _ := Parse(raw, "com.example.node")
	if !nodeErr.IsError() {
		t.Error("NodeError should report IsError")
	}
}

func TestAllHubErrors_CaptureIsSPOREHub(t *testing.T) {
	// SPEC §6.5: Hub-produced errors have capture=SPORE.hub
	original, _ := Parse("clock.get_time ~h1", "com.example.agent")

	castErr := NewError(original, ErrorCodeGeneric, "err")
	if castErr.Capture() != "SPORE.hub" {
		t.Errorf("CastError capture should be SPORE.hub, got %q", castErr.Capture())
	}

	capOriginal, _ := Parse("~h1:clock.get_time result=ok", "com.example.clock")
	capErr := NewError(capOriginal, ErrorCodeGeneric, "err")
	if capErr.Capture() != "SPORE.hub" {
		t.Errorf("CaptureError capture should be SPORE.hub, got %q", capErr.Capture())
	}

	sporeOriginal, _ := Parse("SPORE.node.list ~m1", "com.example.agent")
	sporeErr := NewError(sporeOriginal, ErrorCodeGeneric, "err")
	if sporeErr.Capture() != "SPORE.hub" {
		t.Errorf("SporeError capture should be SPORE.hub, got %q", sporeErr.Capture())
	}
}
