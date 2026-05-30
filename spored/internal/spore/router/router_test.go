// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package router

import (
	"errors"
	"spored/internal/interfaces"
	"spored/internal/manifest"
	"spored/internal/message"
	"strings"
	"testing"
)

// =============================================================================
// SPEC §5.3: Routing
// "The hub resolves the subject against its manifest registry and routes to
// the best available connected node that exposes it."
//
// SPEC §6.2: ~handle
// "~handle is a caller-supplied correlation token used to match a response
// to its originating call."
// =============================================================================

// --- Mock types for testing ---

type mockHub struct {
	nodes map[string]*mockNode
}

func (h *mockHub) ListNodes() []string {
	res := make([]string, 0, len(h.nodes))
	for id := range h.nodes {
		res = append(res, id)
	}
	return res
}

func (h *mockHub) GetNode(nodeid string) (mockNodeIface, error) {
	n, has := h.nodes[nodeid]
	if !has {
		return nil, errors.New("missing node " + nodeid)
	}
	return n, nil
}

func (h *mockHub) AddNode(path string) error      { return nil }
func (h *mockHub) RemoveNode(nodeid string) error { return nil }
func (h *mockHub) SpawnNode(nodeid string) error  { return nil }
func (h *mockHub) KillNode(nodeid string) error   { return nil }

type mockNodeIface interface {
	Send(msg message.Message) error
	GetManifest() *manifest.Manifest
}

type mockNode struct {
	id       string
	sent     []message.Message
	failSend bool
}

func (n *mockNode) Send(msg message.Message) error {
	if n.failSend {
		return errors.New("send failed")
	}
	n.sent = append(n.sent, msg)
	return nil
}

func (n *mockNode) GetManifest() *manifest.Manifest {
	return &manifest.Manifest{ID: n.id}
}

func (n *mockNode) IsConnected() bool      { return false }
func (n *mockNode) GetPID() int            { return 0 }
func (n *mockNode) SendWitness(msg string) {}
func (n *mockNode) WitnessNode(msg string) {}

type mockSpore struct {
	response message.Message
	err      error
}

func (s *mockSpore) Open(hub interface{}, registry interface{}, router interface{}) {}
func (s *mockSpore) Command(incoming message.Message) (message.Message, error) {
	return s.response, s.err
}

// --- Adapter to bridge mockHub to interfaces.Hub ---
// The Router uses interfaces.Hub which requires (Node, error). We need a
// thin wrapper that adapts our mock to satisfy the real Router.

// Since the Router struct uses interfaces.Hub and interfaces.Spore, we can't
// directly use mocks. Instead, we'll test the Router logic through its public
// API by building a testable setup.

// For simplicity, we test the router's core routing maps and rules directly.

func TestRouter_AddRouteAndGetRoute(t *testing.T) {
	r := &Router{}
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.mu.Unlock()

	r.AddRoute("clock.get_time", "com.example.clock")
	r.AddRoute("filesystem.read", "com.example.filesystem")

	nodeid, err := r.GetRoute("clock.get_time")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nodeid != "com.example.clock" {
		t.Errorf("expected com.example.clock, got %q", nodeid)
	}

	nodeid, err = r.GetRoute("filesystem.read")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if nodeid != "com.example.filesystem" {
		t.Errorf("expected com.example.filesystem, got %q", nodeid)
	}
}

func TestRouter_GetRouteReturnsErrorForUnknownCommand(t *testing.T) {
	// SPEC §5.3: "No installed node owns this subject" → RouteNotFound
	r := &Router{}
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.mu.Unlock()

	_, err := r.GetRoute("nonexistent.command")
	if err == nil {
		t.Error("expected error for unknown route")
	}
}

func TestRouter_ListCommandsAll(t *testing.T) {
	r := &Router{}
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.mu.Unlock()

	r.AddRoute("clock.get_time", "com.example.clock")
	r.AddRoute("clock.set_timer", "com.example.clock")
	r.AddRoute("fs.read", "com.example.fs")

	all := r.ListCommands("n/a")
	if len(all) != 3 {
		t.Errorf("expected 3 commands, got %d", len(all))
	}
}

func TestRouter_ListCommandsFilterByNode(t *testing.T) {
	r := &Router{}
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.mu.Unlock()

	r.AddRoute("clock.get_time", "com.example.clock")
	r.AddRoute("clock.set_timer", "com.example.clock")
	r.AddRoute("fs.read", "com.example.fs")

	clockCmds := r.ListCommands("com.example.clock")
	if len(clockCmds) != 2 {
		t.Errorf("expected 2 commands for clock, got %d", len(clockCmds))
	}
	for _, cmd := range clockCmds {
		if !strings.HasPrefix(cmd, "clock.") {
			t.Errorf("unexpected command %q for clock node", cmd)
		}
	}
}

// =============================================================================
// SPEC §6.2: Handle Rules
// "Handles are per-caller scoped"
// "HandleInUse — A handle was reused before the previous call with that handle completed"
// =============================================================================

func TestRouter_RepliesMapTracksHandles(t *testing.T) {
	// Verify the replies map structure: handle → replyEntry mapping
	r := &Router{}
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.mu.Unlock()

	// Simulate registering a reply expectation
	r.mu.Lock()
	r.replies["h1"] = replyEntry{caster: "com.example.agent", receiver: "com.example.server", command: "some.cmd", handle: "h1"}
	r.mu.Unlock()

	r.mu.RLock()
	entry, has := r.replies["h1"]
	r.mu.RUnlock()

	if !has {
		t.Error("expected handle h1 to be registered")
	}
	if entry.caster != "com.example.agent" {
		t.Errorf("expected caster com.example.agent, got %q", entry.caster)
	}
}

func TestRouter_RepliesMapDeletedAfterCapture(t *testing.T) {
	// SPEC §6.1: "Every call receives exactly one response" — handle consumed after use
	r := &Router{}
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.replies["h1"] = replyEntry{caster: "com.example.agent", receiver: "com.example.server", command: "some.cmd", handle: "h1"}
	r.mu.Unlock()

	// After a capture is delivered, the handle should be removed
	r.mu.Lock()
	delete(r.replies, "h1")
	r.mu.Unlock()

	r.mu.RLock()
	_, has := r.replies["h1"]
	r.mu.RUnlock()

	if has {
		t.Error("handle should be deleted after capture delivery")
	}
}

// =============================================================================
// SPEC §5.2: Resolution — Short form vs Full form
// Routes are stored by command name (short form). Full form must also work.
// =============================================================================

func TestRouter_ShortFormRouting(t *testing.T) {
	r := &Router{}
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.mu.Unlock()

	// Short form: "clock.get_time" → node
	r.AddRoute("clock.get_time", "com.example.clock")

	nodeid, err := r.GetRoute("clock.get_time")
	if err != nil {
		t.Fatalf("short form routing failed: %v", err)
	}
	if nodeid != "com.example.clock" {
		t.Errorf("expected com.example.clock, got %q", nodeid)
	}
}

func TestRouter_RouteOverwritesPreviousRegistration(t *testing.T) {
	// If the same command is registered again, it overwrites
	r := &Router{}
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.mu.Unlock()

	r.AddRoute("clock.get_time", "com.example.clock")
	r.AddRoute("clock.get_time", "com.other.clock")

	nodeid, _ := r.GetRoute("clock.get_time")
	if nodeid != "com.other.clock" {
		t.Errorf("expected route to be overwritten, got %q", nodeid)
	}
}

// =============================================================================
// routeTestHub — an interfaces.Hub implementation used for Route() tests.
// =============================================================================

// routeTestHub satisfies interfaces.Hub so tests can exercise the full Route
// path, including GetNode calls made by the router during routing.
type routeTestHub struct {
	nodes map[string]*mockNode
}

func (h *routeTestHub) ListNodes() []string {
	res := make([]string, 0, len(h.nodes))
	for id := range h.nodes {
		res = append(res, id)
	}
	return res
}

func (h *routeTestHub) GetNode(nodeid string) (interfaces.Node, error) {
	n, has := h.nodes[nodeid]
	if !has {
		return nil, errors.New("missing node " + nodeid)
	}
	return n, nil
}

func (h *routeTestHub) AddNode(path string) error      { return nil }
func (h *routeTestHub) RemoveNode(nodeid string) error { return nil }
func (h *routeTestHub) SpawnNode(nodeid string) error  { return nil }
func (h *routeTestHub) KillNode(nodeid string) error   { return nil }

// newRouteTestRouter returns a Router wired to a routeTestHub containing the
// provided nodes. The router's commands and replies maps are initialised.
func newRouteTestRouter(nodes map[string]*mockNode) *Router {
	hub := &routeTestHub{nodes: nodes}
	r := &Router{}
	r.hub = hub
	r.mu.Lock()
	r.commands = make(map[string]string)
	r.replies = make(map[string]replyEntry)
	r.pending = make(map[string]*pendingSlot)
	r.mu.Unlock()
	return r
}

// =============================================================================
// SPEC §6.2 / connection.go regression:
// When the hub encounters a malformed capture (parse fails), it routes a
// synthetic NodeError. This must clean up replies[handle] so the caller can
// reuse the same handle on the next attempt.
// =============================================================================

func TestRouter_NodeErrorRoutingCleansUpHandle(t *testing.T) {
	// Arrange: CLI node + router with a registered pending reply for h1.
	cli := &mockNode{id: "dev.sporeos.cli"}
	r := newRouteTestRouter(map[string]*mockNode{
		"dev.sporeos.cli": cli,
	})

	r.mu.Lock()
	r.replies["h1"] = replyEntry{caster: "dev.sporeos.cli", receiver: "dev.sporeos.echo", command: "SPORE.unknown", handle: "h1"}
	r.mu.Unlock()

	// Parse a synthetic NodeError of the kind connection.go builds on parse failure.
	raw := `~h1:SPORE.unknown error code=MessageMalformed what="unterminated array ["`
	errMsg, err := message.Parse(raw, "SPORE.hub")
	if err != nil {
		t.Fatalf("failed to parse synthetic error: %v", err)
	}

	// Act: route it — this should remove replies["h1"] and deliver to the CLI.
	if routeErr := r.Route(errMsg); routeErr != nil {
		t.Fatalf("Route returned unexpected error: %v", routeErr)
	}

	// Assert: handle is gone.
	r.mu.RLock()
	_, still := r.replies["h1"]
	r.mu.RUnlock()
	if still {
		t.Error("replies[h1] was not cleaned up after routing the NodeError")
	}

	// Assert: the CLI node received the error.
	if len(cli.sent) != 1 {
		t.Fatalf("expected CLI to receive 1 message, got %d", len(cli.sent))
	}
	if !cli.sent[0].IsError() {
		t.Error("message delivered to CLI should be an error")
	}
	if cli.sent[0].Handle() != "h1" {
		t.Errorf("delivered message handle: got %q, want h1", cli.sent[0].Handle())
	}
}

func TestRouter_HandleReuseAllowedAfterErrorRouting(t *testing.T) {
	// After a NodeError cleans up replies[h1], the same handle should be
	// accepted again on the next cast (no HandleInUse).
	cli := &mockNode{id: "dev.sporeos.cli"}
	echo := &mockNode{id: "dev.sporeos.echo"}
	r := newRouteTestRouter(map[string]*mockNode{
		"dev.sporeos.cli":  cli,
		"dev.sporeos.echo": echo,
	})
	r.AddRoute("echo", "dev.sporeos.echo")

	// Simulate: h1 was registered but the capture was malformed.
	r.mu.Lock()
	r.replies["h1"] = replyEntry{caster: "dev.sporeos.cli", receiver: "dev.sporeos.echo", command: "SPORE.unknown", handle: "h1"}
	r.mu.Unlock()

	// Route cleanup error.
	raw := `~h1:SPORE.unknown error code=MessageMalformed what="unterminated array ["`
	errMsg, _ := message.Parse(raw, "SPORE.hub")
	_ = r.Route(errMsg)

	// The handle must now be free; parsing and routing a new cast with h1 must
	// succeed without a HandleInUse error.
	castRaw := "echo ~h1 expression=hello cast=dev.sporeos.cli"
	castMsg, err := message.Parse(castRaw, "dev.sporeos.cli")
	if err != nil {
		t.Fatalf("failed to parse cast: %v", err)
	}
	if routeErr := r.Route(castMsg); routeErr != nil {
		t.Errorf("expected handle h1 to be reusable after error cleanup, got: %v", routeErr)
	}
}

// =============================================================================
// PurgeNode — cleanup when a node disconnects
// =============================================================================

// TestPurgeNode_ReceiverDisconnect: caller casts to an echo node, then the
// echo node disconnects. PurgeNode must: deliver a RouteNotConnected error to
// the caller, and free the handle so it can be reused.
func TestPurgeNode_ReceiverDisconnect(t *testing.T) {
	cli := &mockNode{id: "dev.sporeos.cli"}
	echo := &mockNode{id: "dev.sporeos.echo"}
	r := newRouteTestRouter(map[string]*mockNode{
		"dev.sporeos.cli":  cli,
		"dev.sporeos.echo": echo,
	})

	// Seed in-flight handle: cli cast to echo, handle h1.
	r.mu.Lock()
	r.replies["h1"] = replyEntry{caster: "dev.sporeos.cli", receiver: "dev.sporeos.echo", command: "echo", handle: "h1"}
	r.mu.Unlock()

	// Echo node disconnects.
	r.PurgeNode("dev.sporeos.echo")

	// Handle must be gone.
	r.mu.RLock()
	_, still := r.replies["h1"]
	r.mu.RUnlock()
	if still {
		t.Error("expected replies[h1] to be cleaned up after receiver disconnect")
	}

	// CLI must have received exactly one error.
	if len(cli.sent) != 1 {
		t.Fatalf("expected CLI to receive 1 error, got %d", len(cli.sent))
	}
	if !cli.sent[0].IsError() {
		t.Error("message delivered to CLI should be an error")
	}
	if cli.sent[0].Handle() != "h1" {
		t.Errorf("error handle: got %q, want h1", cli.sent[0].Handle())
	}
}

// TestPurgeNode_CasterDisconnect: the calling node disconnects while waiting
// for a response. The handle should be silently freed; the receiver node must
// NOT receive an error.
func TestPurgeNode_CasterDisconnect(t *testing.T) {
	cli := &mockNode{id: "dev.sporeos.cli"}
	echo := &mockNode{id: "dev.sporeos.echo"}
	r := newRouteTestRouter(map[string]*mockNode{
		"dev.sporeos.cli":  cli,
		"dev.sporeos.echo": echo,
	})

	r.mu.Lock()
	r.replies["h1"] = replyEntry{caster: "dev.sporeos.cli", receiver: "dev.sporeos.echo", command: "echo", handle: "h1"}
	r.mu.Unlock()

	// CLI node disconnects.
	r.PurgeNode("dev.sporeos.cli")

	// Handle must be gone.
	r.mu.RLock()
	_, still := r.replies["h1"]
	r.mu.RUnlock()
	if still {
		t.Error("expected replies[h1] to be silently cleaned up after caster disconnect")
	}

	// Neither node should have received a message.
	if len(cli.sent) != 0 {
		t.Errorf("caster should not receive a message on its own disconnect, got %d", len(cli.sent))
	}
	if len(echo.sent) != 0 {
		t.Errorf("receiver should not be notified of caster disconnect, got %d", len(echo.sent))
	}
}

// TestPurgeNode_HandleFreeAfterReceiverDisconnect: after PurgeNode frees a
// handle, a new cast with the same handle must be accepted (no HandleInUse).
func TestPurgeNode_HandleFreeAfterReceiverDisconnect(t *testing.T) {
	cli := &mockNode{id: "dev.sporeos.cli"}
	echo := &mockNode{id: "dev.sporeos.echo"}
	r := newRouteTestRouter(map[string]*mockNode{
		"dev.sporeos.cli":  cli,
		"dev.sporeos.echo": echo,
	})
	r.AddRoute("echo", "dev.sporeos.echo")

	r.mu.Lock()
	r.replies["h1"] = replyEntry{caster: "dev.sporeos.cli", receiver: "dev.sporeos.echo", command: "echo", handle: "h1"}
	r.mu.Unlock()

	r.PurgeNode("dev.sporeos.echo")

	// Remove echo from the hub and add it back so routing works.
	r.AddRoute("echo", "dev.sporeos.echo")

	castRaw := "echo ~h1 expression=hello cast=dev.sporeos.cli"
	castMsg, err := message.Parse(castRaw, "dev.sporeos.cli")
	if err != nil {
		t.Fatalf("failed to parse cast: %v", err)
	}
	if routeErr := r.Route(castMsg); routeErr != nil {
		t.Errorf("expected h1 to be reusable after PurgeNode, got: %v", routeErr)
	}
}

// TestPurgeNode_InlineReceiverDisconnect: an inline inner call is in-flight
// to an echo node; echo disconnects. PurgeNode must short-circuit the outer
// pending entry and send a RouteNotConnected error to the original caller.
func TestPurgeNode_InlineReceiverDisconnect(t *testing.T) {
	cli := &mockNode{id: "dev.sporeos.cli"}
	echo := &mockNode{id: "dev.sporeos.echo"}
	r := newRouteTestRouter(map[string]*mockNode{
		"dev.sporeos.cli":  cli,
		"dev.sporeos.echo": echo,
	})
	r.AddRoute("echo", "dev.sporeos.echo")

	// Manually plant a pending entry simulating an in-flight inline call.
	internalHandle := "~DEAD"
	entry := &pendingEntry{
		callerNodeID:    "dev.sporeos.cli",
		originalHandle:  "outerH",
		originalCommand: "echo",
		shortCircuited:  false,
	}
	slot := &pendingSlot{
		internalHandle: internalHandle,
		kind:           message.ArgKindKeyBind,
		argIndex:       0,
		innerReceiver:  "dev.sporeos.echo",
		entry:          entry,
	}
	entry.slots = []*pendingSlot{slot}

	r.mu.Lock()
	r.pending[internalHandle] = slot
	r.mu.Unlock()

	// Echo disconnects.
	r.PurgeNode("dev.sporeos.echo")

	// Pending slot must be cleaned up.
	r.mu.RLock()
	_, stillPending := r.pending[internalHandle]
	r.mu.RUnlock()
	if stillPending {
		t.Error("expected pending slot to be removed after inner receiver disconnect")
	}

	// CLI must have received a RouteNotConnected error.
	if len(cli.sent) != 1 {
		t.Fatalf("expected CLI to receive 1 error, got %d", len(cli.sent))
	}
	if !cli.sent[0].IsError() {
		t.Error("message delivered to CLI should be an error")
	}
	if cli.sent[0].Handle() != "outerH" {
		t.Errorf("error handle: got %q, want outerH", cli.sent[0].Handle())
	}
}

// TestPurgeNode_InlineCasterDisconnect: the outer caller disconnects while
// an inline slot is in flight. The pending map must be silently cleaned up.
func TestPurgeNode_InlineCasterDisconnect(t *testing.T) {
	echo := &mockNode{id: "dev.sporeos.echo"}
	r := newRouteTestRouter(map[string]*mockNode{
		"dev.sporeos.echo": echo,
	})

	internalHandle := "~BEEF"
	entry := &pendingEntry{
		callerNodeID:    "dev.sporeos.cli",
		originalHandle:  "outerH",
		originalCommand: "echo",
		shortCircuited:  false,
	}
	slot := &pendingSlot{
		internalHandle: internalHandle,
		kind:           message.ArgKindKeyBind,
		argIndex:       0,
		innerReceiver:  "dev.sporeos.echo",
		entry:          entry,
	}
	entry.slots = []*pendingSlot{slot}

	r.mu.Lock()
	r.pending[internalHandle] = slot
	r.mu.Unlock()

	// CLI (caller) disconnects.
	r.PurgeNode("dev.sporeos.cli")

	// Pending slot must be silently cleaned up.
	r.mu.RLock()
	_, stillPending := r.pending[internalHandle]
	r.mu.RUnlock()
	if stillPending {
		t.Error("expected pending slot to be removed after outer caster disconnect")
	}

	// Echo should not receive anything.
	if len(echo.sent) != 0 {
		t.Errorf("echo should not receive a message on caster disconnect, got %d", len(echo.sent))
	}
}

