// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package router

// =============================================================================
// Inline call resolution tests — Phase 2/3/4
//
// SPEC §6.4: Inline Call Substitution
//
// Test approach: synchronous in-process simulation.
//  1. Call r.Route(outerCast) — fires inner casts to mock inner nodes.
//  2. Inspect what the inner node received, craft a response.
//  3. Call r.Route(innerResponse) — completes the slot(s), fires the
//     resolved outer call to the final node.
//  4. Assert what the caller/final node received.
//
// Hub-internal handles have the form ~~XXXX on the wire. After Capture.Parse
// strips the outer ~, the handle value stored in r.pending is "~XXXX".
// =============================================================================

import (
	"spored/internal/message"
	"strings"
	"testing"
)

// newInlineRouter builds a fully wired Router for inline tests.
// nodes is the full node map (caller + inner targets + final target).
// commands maps command → nodeID.
func newInlineRouter(nodes map[string]*mockNode, commands map[string]string) *Router {
	r := newRouteTestRouter(nodes)
	for cmd, nid := range commands {
		r.AddRoute(cmd, nid)
	}
	return r
}

// internalHandleFrom extracts the internal handle value (e.g. "~2BD4") from a
// message that was sent to an inner node. The message's handle token is ~~XXXX
// on the wire; after Cast.Parse strips the outer ~, Handle() returns "~XXXX".
func internalHandleFrom(msg message.Message) string {
	return msg.Handle()
}

// buildCapture constructs a response string that simulates a node replying
// to an inner cast with the given handle value. `from` is the node's id.
// Additional key=value pairs can be appended in `fields`.
func buildCapture(handleVal, command string, fields ...string) string {
	// Wire handle token = "~" + handleVal  ("~" + "~XXXX" = "~~XXXX")
	wireHandle := "~" + handleVal
	parts := append([]string{wireHandle + ":" + command}, fields...)
	return strings.Join(parts, " ")
}

// =============================================================================
// §6.4 Form 1 — Key-binding: key=(subject [args...])
// =============================================================================

func TestInline_KeyBind_SingleOutput(t *testing.T) {
	// SPORE.node.install path=(dialog.file_picker) ~h1
	// Inner: dialog.file_picker responds with path="/chosen/path"  (one data field)
	// Final: SPORE.node.install is not registered, but we check what the resolver does.

	caller := &mockNode{id: "com.example.agent"}
	picker := &mockNode{id: "com.example.dialog"}
	installer := &mockNode{id: "com.example.installer"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent":     caller,
			"com.example.dialog":    picker,
			"com.example.installer": installer,
		},
		map[string]string{
			"dialog.file_picker": "com.example.dialog",
			"installer.add":      "com.example.installer",
		},
	)

	// 1. Route the outer cast.
	raw := "installer.add path=(dialog.file_picker) ~h1"
	outerCast, err := message.Parse(raw, "com.example.agent")
	if err != nil {
		t.Fatalf("Parse outer cast: %v", err)
	}

	if err := r.Route(outerCast); err != nil {
		t.Fatalf("Route outer cast: %v", err)
	}

	// 2. Inner node (picker) should have received one inner cast.
	if len(picker.sent) != 1 {
		t.Fatalf("picker: expected 1 message, got %d", len(picker.sent))
	}
	innerMsg := picker.sent[0]
	if innerMsg.Command() != "dialog.file_picker" {
		t.Errorf("inner command: got %q", innerMsg.Command())
	}

	// Inner handle is ~~XXXX; Handle() returns "~XXXX".
	innerHandle := internalHandleFrom(innerMsg)
	if !strings.HasPrefix(innerHandle, "~") {
		t.Errorf("expected hub-internal handle (~~XXXX form), got %q", innerHandle)
	}

	// 3. Simulate picker responding with one field.
	captureRaw := buildCapture(innerHandle, "dialog.file_picker", `path="/chosen/path"`)
	captureMsg, err := message.Parse(captureRaw, "com.example.dialog")
	if err != nil {
		t.Fatalf("Parse inner capture: %v", err)
	}

	if err := r.Route(captureMsg); err != nil {
		t.Fatalf("Route inner capture: %v", err)
	}

	// 4. Final node should have received the resolved outer call.
	if len(installer.sent) != 1 {
		t.Fatalf("installer: expected 1 message, got %d", len(installer.sent))
	}
	finalMsg := installer.sent[0]
	if finalMsg.Command() != "installer.add" {
		t.Errorf("final command: got %q", finalMsg.Command())
	}
	raw = finalMsg.ToString()
	if !strings.Contains(raw, `path="/chosen/path"`) {
		t.Errorf("resolved outer call missing path field; got: %s", raw)
	}
	if strings.Contains(raw, "(") {
		t.Errorf("resolved outer call should not contain inline parens; got: %s", raw)
	}
}

func TestInline_KeyBind_MultiOutput_NameMatch(t *testing.T) {
	// filesystem.read path=(shortpath project=notes) ~h1
	// Inner: shortpath responds with  path=/notes  name=notes  (two fields)
	// Outer key is "path" → must pick "path" field.

	caller := &mockNode{id: "com.example.agent"}
	shortpath := &mockNode{id: "com.example.shortpath"}
	fsNode := &mockNode{id: "com.example.fs"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent":     caller,
			"com.example.shortpath": shortpath,
			"com.example.fs":        fsNode,
		},
		map[string]string{
			"shortpath":       "com.example.shortpath",
			"filesystem.read": "com.example.fs",
		},
	)

	raw := "filesystem.read path=(shortpath project=notes) ~h1"
	outerCast, _ := message.Parse(raw, "com.example.agent")
	if err := r.Route(outerCast); err != nil {
		t.Fatalf("Route outer cast: %v", err)
	}
	if len(shortpath.sent) != 1 {
		t.Fatalf("shortpath: expected 1 message, got %d", len(shortpath.sent))
	}

	innerHandle := internalHandleFrom(shortpath.sent[0])
	captureRaw := buildCapture(innerHandle, "shortpath", "path=/notes", "name=notes")
	captureMsg, _ := message.Parse(captureRaw, "com.example.shortpath")
	if err := r.Route(captureMsg); err != nil {
		t.Fatalf("Route inner capture: %v", err)
	}

	if len(fsNode.sent) != 1 {
		t.Fatalf("fs: expected 1 message, got %d", len(fsNode.sent))
	}
	resolved := fsNode.sent[0].ToString()
	if !strings.Contains(resolved, "path=/notes") {
		t.Errorf("resolved call should have path=/notes; got: %s", resolved)
	}
	// name= must NOT appear (it's the inner node's output, not the outer key)
	if strings.Contains(resolved, "name=notes") {
		t.Errorf("resolved call should not carry inner-only field name=notes; got: %s", resolved)
	}
}

func TestInline_KeyBind_MultiOutput_NoNameMatch(t *testing.T) {
	// Outer key is "path" but inner responds with "bar" and "baz" — error.

	caller := &mockNode{id: "com.example.agent"}
	inner := &mockNode{id: "com.example.inner"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent": caller,
			"com.example.inner": inner,
		},
		map[string]string{"some.cmd": "com.example.inner"},
	)

	raw := "filesystem.read path=(some.cmd) ~h1"
	outerCast, _ := message.Parse(raw, "com.example.agent")

	// filesystem.read is not registered, but the inline fires some.cmd first.
	// Register filesystem.read too (even though it won't be reached).
	r.AddRoute("filesystem.read", "com.example.inner")

	if err := r.Route(outerCast); err != nil {
		t.Fatalf("Route outer cast: %v", err)
	}
	if len(inner.sent) != 1 {
		t.Fatalf("inner: expected 1 message, got %d", len(inner.sent))
	}

	innerHandle := internalHandleFrom(inner.sent[0])
	captureRaw := buildCapture(innerHandle, "some.cmd", "bar=1", "baz=2")
	captureMsg, _ := message.Parse(captureRaw, "com.example.inner")
	if err := r.Route(captureMsg); err != nil {
		t.Fatalf("Route inner capture (error path): %v", err)
	}

	// Caller must receive an error.
	if len(caller.sent) != 1 {
		t.Fatalf("caller: expected 1 message (error), got %d", len(caller.sent))
	}
	errMsg := caller.sent[0]
	if !errMsg.IsError() {
		t.Errorf("caller should have received an error; got: %s", errMsg.ToString())
	}
}

// =============================================================================
// §6.4 Form 2 — Spread: (subject [args...])
// =============================================================================

func TestInline_Spread(t *testing.T) {
	// filesystem.list (dialog.dir.open) recursive ~h1
	// Inner: dialog.dir.open responds with  path=/projects  filter=*.go
	// Resolved: filesystem.list path=/projects filter=*.go recursive ~h1

	caller := &mockNode{id: "com.example.agent"}
	dialog := &mockNode{id: "com.example.dialog"}
	fsNode := &mockNode{id: "com.example.fs"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent":  caller,
			"com.example.dialog": dialog,
			"com.example.fs":     fsNode,
		},
		map[string]string{
			"dialog.dir.open":  "com.example.dialog",
			"filesystem.list": "com.example.fs",
		},
	)

	raw := "filesystem.list (dialog.dir.open) recursive ~h1"
	outerCast, err := message.Parse(raw, "com.example.agent")
	if err != nil {
		t.Fatalf("Parse outer cast: %v", err)
	}
	if err := r.Route(outerCast); err != nil {
		t.Fatalf("Route outer cast: %v", err)
	}
	if len(dialog.sent) != 1 {
		t.Fatalf("dialog: expected 1 message, got %d", len(dialog.sent))
	}

	innerHandle := internalHandleFrom(dialog.sent[0])
	captureRaw := buildCapture(innerHandle, "dialog.dir.open", "path=/projects", "filter=*.go")
	captureMsg, _ := message.Parse(captureRaw, "com.example.dialog")
	if err := r.Route(captureMsg); err != nil {
		t.Fatalf("Route inner capture: %v", err)
	}

	if len(fsNode.sent) != 1 {
		t.Fatalf("fs: expected 1 message, got %d", len(fsNode.sent))
	}
	resolved := fsNode.sent[0].ToString()
	if !strings.Contains(resolved, "path=/projects") {
		t.Errorf("spread: expected path=/projects in resolved call; got: %s", resolved)
	}
	if !strings.Contains(resolved, "filter=*.go") {
		t.Errorf("spread: expected filter=*.go in resolved call; got: %s", resolved)
	}
	if !strings.Contains(resolved, "recursive") {
		t.Errorf("spread: original flag 'recursive' must survive; got: %s", resolved)
	}
	if strings.Contains(resolved, "(") {
		t.Errorf("spread: resolved call must not contain inline parens; got: %s", resolved)
	}
}

func TestInline_SpreadAndKeyBind(t *testing.T) {
	// cmd.do (inner.spread) key=(inner.bind) ~h1
	// Both resolve: spread provides x=1 y=2, key-bind provides key=val
	// Resolved: cmd.do x=1 y=2 key=val ~h1

	caller := &mockNode{id: "com.example.agent"}
	spreadNode := &mockNode{id: "com.example.spread"}
	bindNode := &mockNode{id: "com.example.bind"}
	target := &mockNode{id: "com.example.target"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent":  caller,
			"com.example.spread": spreadNode,
			"com.example.bind":   bindNode,
			"com.example.target": target,
		},
		map[string]string{
			"inner.spread": "com.example.spread",
			"inner.bind":   "com.example.bind",
			"cmd.do":       "com.example.target",
		},
	)

	raw := "cmd.do (inner.spread) key=(inner.bind) ~h1"
	outerCast, err := message.Parse(raw, "com.example.agent")
	if err != nil {
		t.Fatalf("Parse outer cast: %v", err)
	}
	if err := r.Route(outerCast); err != nil {
		t.Fatalf("Route outer cast: %v", err)
	}

	// Both inner nodes should have received a cast.
	if len(spreadNode.sent) != 1 {
		t.Fatalf("spreadNode: expected 1 message, got %d", len(spreadNode.sent))
	}
	if len(bindNode.sent) != 1 {
		t.Fatalf("bindNode: expected 1 message, got %d", len(bindNode.sent))
	}

	// Resolve spread first.
	spreadHandle := internalHandleFrom(spreadNode.sent[0])
	spreadCapture, _ := message.Parse(
		buildCapture(spreadHandle, "inner.spread", "x=1", "y=2"),
		"com.example.spread",
	)
	if err := r.Route(spreadCapture); err != nil {
		t.Fatalf("Route spread capture: %v", err)
	}
	// target must not have been called yet (bind slot still pending).
	if len(target.sent) != 0 {
		t.Fatalf("target should not have been called yet; got %d messages", len(target.sent))
	}

	// Resolve key-bind.
	bindHandle := internalHandleFrom(bindNode.sent[0])
	bindCapture, _ := message.Parse(
		buildCapture(bindHandle, "inner.bind", "key=val"),
		"com.example.bind",
	)
	if err := r.Route(bindCapture); err != nil {
		t.Fatalf("Route bind capture: %v", err)
	}

	// Now the target should have the resolved outer call.
	if len(target.sent) != 1 {
		t.Fatalf("target: expected 1 message, got %d", len(target.sent))
	}
	resolved := target.sent[0].ToString()
	if !strings.Contains(resolved, "x=1") {
		t.Errorf("spread field x=1 missing; got: %s", resolved)
	}
	if !strings.Contains(resolved, "y=2") {
		t.Errorf("spread field y=2 missing; got: %s", resolved)
	}
	if !strings.Contains(resolved, "key=val") {
		t.Errorf("key-bind field key=val missing; got: %s", resolved)
	}
}

// =============================================================================
// §6.4 Cancelled and error propagation
// =============================================================================

func TestInline_InnerCancelled(t *testing.T) {
	// Inner call responds with cancelled → outer call aborted, caller gets cancelled.

	caller := &mockNode{id: "com.example.agent"}
	dialog := &mockNode{id: "com.example.dialog"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent":  caller,
			"com.example.dialog": dialog,
		},
		map[string]string{
			"dialog.file_picker": "com.example.dialog",
			"filesystem.read":    "com.example.dialog", // fallback so outer cast parses
		},
	)

	raw := "filesystem.read path=(dialog.file_picker) ~h1"
	outerCast, _ := message.Parse(raw, "com.example.agent")
	if err := r.Route(outerCast); err != nil {
		t.Fatalf("Route outer cast: %v", err)
	}

	innerHandle := internalHandleFrom(dialog.sent[0])
	// Build a cancelled response for the inner call.
	cancelledRaw := "~" + innerHandle + ":dialog.file_picker cancelled"
	cancelledMsg, err := message.Parse(cancelledRaw, "com.example.dialog")
	if err != nil {
		t.Fatalf("Parse cancelled: %v", err)
	}
	if err := r.Route(cancelledMsg); err != nil {
		t.Fatalf("Route cancelled: %v", err)
	}

	if len(caller.sent) != 1 {
		t.Fatalf("caller: expected 1 message (cancelled), got %d", len(caller.sent))
	}
	if !caller.sent[0].IsCancelled() {
		t.Errorf("caller should have received cancelled; got: %s", caller.sent[0].ToString())
	}
	if caller.sent[0].Handle() != "h1" {
		t.Errorf("cancelled must carry original outer handle h1; got %q", caller.sent[0].Handle())
	}
}

func TestInline_InnerError(t *testing.T) {
	// Inner call responds with error → outer call aborted, caller gets error.

	caller := &mockNode{id: "com.example.agent"}
	inner := &mockNode{id: "com.example.inner"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent": caller,
			"com.example.inner": inner,
		},
		map[string]string{
			"some.cmd":        "com.example.inner",
			"filesystem.read": "com.example.inner",
		},
	)

	raw := "filesystem.read path=(some.cmd) ~h1"
	outerCast, _ := message.Parse(raw, "com.example.agent")
	if err := r.Route(outerCast); err != nil {
		t.Fatalf("Route outer cast: %v", err)
	}

	innerHandle := internalHandleFrom(inner.sent[0])
	errorRaw := `~` + innerHandle + `:some.cmd error node_error code=Runtime what="disk failure"`
	errorMsg, err := message.Parse(errorRaw, "com.example.inner")
	if err != nil {
		t.Fatalf("Parse inner error: %v", err)
	}
	if err := r.Route(errorMsg); err != nil {
		t.Fatalf("Route inner error: %v", err)
	}

	if len(caller.sent) != 1 {
		t.Fatalf("caller: expected 1 message (error), got %d", len(caller.sent))
	}
	if !caller.sent[0].IsError() {
		t.Errorf("caller should have received error; got: %s", caller.sent[0].ToString())
	}
	if caller.sent[0].Handle() != "h1" {
		t.Errorf("error must carry original outer handle h1; got %q", caller.sent[0].Handle())
	}
}

func TestInline_TwoSpreads_Error(t *testing.T) {
	// Spec §6.4: only one spread per outer call. Two spreads → error before any inner cast fires.

	caller := &mockNode{id: "com.example.agent"}
	inner := &mockNode{id: "com.example.inner"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent": caller,
			"com.example.inner": inner,
		},
		map[string]string{
			"a.call":      "com.example.inner",
			"b.call":      "com.example.inner",
			"filesystem.list": "com.example.inner",
		},
	)

	raw := "filesystem.list (a.call) (b.call) ~h1"
	outerCast, err := message.Parse(raw, "com.example.agent")
	if err != nil {
		t.Fatalf("Parse outer cast: %v", err)
	}

	routeErr := r.Route(outerCast)
	// routeInline returns a non-nil error AND sends error to caller.
	if routeErr == nil {
		t.Error("expected error from Route for two spread forms")
	}

	// Caller must have received an error message.
	if len(caller.sent) != 1 {
		t.Fatalf("caller: expected 1 error message, got %d", len(caller.sent))
	}
	if !caller.sent[0].IsError() {
		t.Errorf("caller should have received error; got: %s", caller.sent[0].ToString())
	}

	// No inner casts must have been sent.
	if len(inner.sent) != 0 {
		t.Errorf("no inner casts should fire for two-spread error; got %d", len(inner.sent))
	}
}

// =============================================================================
// Phase 4.2 — Nested inline forms
// =============================================================================

func TestInline_Nested_KeyBind(t *testing.T) {
	// filesystem.list path=(dialog.file_picker path=(project.root)) ~h1
	//
	// Resolution order:
	//  1. Outer fires inner cast: dialog.file_picker path=(project.root) ~~AAAA
	//  2. routeInnerCast detects nested inline, fires: project.root ~~BBBB
	//     to projectNode.
	//  3. project.root responds ~~BBBB:project.root path=/home/user
	//     → inner pending entry resolves → re-routes dialog.file_picker path=/home/user ~~AAAA
	//     → picker node receives dialog.file_picker path=/home/user call
	//  4. picker responds ~~AAAA:dialog.file_picker path="/chosen/path"
	//     → outer pending entry resolves → re-routes filesystem.list path="/chosen/path" ~h1
	//     → fsNode receives the final call.

	caller := &mockNode{id: "com.example.agent"}
	picker := &mockNode{id: "com.example.picker"}
	project := &mockNode{id: "com.example.project"}
	fsNode := &mockNode{id: "com.example.fs"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent":   caller,
			"com.example.picker":  picker,
			"com.example.project": project,
			"com.example.fs":      fsNode,
		},
		map[string]string{
			"dialog.file_picker": "com.example.picker",
			"project.root":       "com.example.project",
			"filesystem.list":    "com.example.fs",
		},
	)

	raw := "filesystem.list path=(dialog.file_picker path=(project.root)) ~h1"
	outerCast, err := message.Parse(raw, "com.example.agent")
	if err != nil {
		t.Fatalf("Parse outer cast: %v", err)
	}

	// Step 1 & 2.
	if err := r.Route(outerCast); err != nil {
		t.Fatalf("Route outer cast: %v", err)
	}

	// project.root should have received the innermost cast.
	if len(project.sent) != 1 {
		t.Fatalf("project: expected 1 message, got %d", len(project.sent))
	}
	innerInnerHandle := internalHandleFrom(project.sent[0])
	if !strings.HasPrefix(innerInnerHandle, "~") {
		t.Errorf("expected hub-internal handle, got %q", innerInnerHandle)
	}

	// Step 3: project.root responds.
	projCaptureRaw := buildCapture(innerInnerHandle, "project.root", "path=/home/user")
	projCapture, err := message.Parse(projCaptureRaw, "com.example.project")
	if err != nil {
		t.Fatalf("Parse project capture: %v", err)
	}
	if err := r.Route(projCapture); err != nil {
		t.Fatalf("Route project capture: %v", err)
	}

	// picker should now have the resolved inner call (no inline forms).
	if len(picker.sent) != 1 {
		t.Fatalf("picker: expected 1 message after project.root resolved, got %d", len(picker.sent))
	}
	pickerMsg := picker.sent[0]
	if pickerMsg.Command() != "dialog.file_picker" {
		t.Errorf("picker: got command %q", pickerMsg.Command())
	}
	if !strings.Contains(pickerMsg.ToString(), "path=/home/user") {
		t.Errorf("picker: expected path=/home/user in call; got: %s", pickerMsg.ToString())
	}
	if strings.Contains(pickerMsg.ToString(), "(") {
		t.Errorf("picker: call should not still contain inline forms; got: %s", pickerMsg.ToString())
	}

	// Step 4: picker responds.
	pickerHandle := internalHandleFrom(pickerMsg)
	pickerCaptureRaw := buildCapture(pickerHandle, "dialog.file_picker", `path="/chosen/path"`)
	pickerCapture, err := message.Parse(pickerCaptureRaw, "com.example.picker")
	if err != nil {
		t.Fatalf("Parse picker capture: %v", err)
	}
	if err := r.Route(pickerCapture); err != nil {
		t.Fatalf("Route picker capture: %v", err)
	}

	// Finally, fsNode should receive the fully resolved outer call.
	if len(fsNode.sent) != 1 {
		t.Fatalf("fs: expected 1 message, got %d", len(fsNode.sent))
	}
	finalRaw := fsNode.sent[0].ToString()
	if !strings.Contains(finalRaw, `path="/chosen/path"`) {
		t.Errorf("fs: expected path=\"/chosen/path\" in resolved call; got: %s", finalRaw)
	}
	if strings.Contains(finalRaw, "(") {
		t.Errorf("fs: resolved call must not contain inline parens; got: %s", finalRaw)
	}
}

// =============================================================================
// Pending map hygiene
// =============================================================================

func TestInline_PendingMapCleanedUpAfterSuccess(t *testing.T) {
	// After successful resolution, r.pending must be empty.

	caller := &mockNode{id: "com.example.agent"}
	picker := &mockNode{id: "com.example.dialog"}
	installer := &mockNode{id: "com.example.installer"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent":     caller,
			"com.example.dialog":    picker,
			"com.example.installer": installer,
		},
		map[string]string{
			"dialog.file_picker": "com.example.dialog",
			"installer.add":      "com.example.installer",
		},
	)

	outerCast, _ := message.Parse("installer.add path=(dialog.file_picker) ~h1", "com.example.agent")
	_ = r.Route(outerCast)

	innerHandle := internalHandleFrom(picker.sent[0])
	captureMsg, _ := message.Parse(
		buildCapture(innerHandle, "dialog.file_picker", "path=/chosen"),
		"com.example.dialog",
	)
	_ = r.Route(captureMsg)

	r.mu.RLock()
	pendingCount := len(r.pending)
	r.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("r.pending should be empty after resolution; got %d entries", pendingCount)
	}
}

func TestInline_PendingMapCleanedUpAfterCancelled(t *testing.T) {
	caller := &mockNode{id: "com.example.agent"}
	inner := &mockNode{id: "com.example.inner"}

	r := newInlineRouter(
		map[string]*mockNode{
			"com.example.agent": caller,
			"com.example.inner": inner,
		},
		map[string]string{
			"some.cmd":        "com.example.inner",
			"filesystem.read": "com.example.inner",
		},
	)

	outerCast, _ := message.Parse("filesystem.read path=(some.cmd) ~h1", "com.example.agent")
	_ = r.Route(outerCast)

	innerHandle := internalHandleFrom(inner.sent[0])
	cancelledRaw := "~" + innerHandle + ":some.cmd cancelled"
	cancelledMsg, _ := message.Parse(cancelledRaw, "com.example.inner")
	_ = r.Route(cancelledMsg)

	r.mu.RLock()
	pendingCount := len(r.pending)
	r.mu.RUnlock()

	if pendingCount != 0 {
		t.Errorf("r.pending should be empty after cancelled; got %d entries", pendingCount)
	}
}
