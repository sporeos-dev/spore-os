// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package router

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"spored/internal/message"
)

// ---- Protocol field filter --------------------------------------------------

// protocolFieldKeys are the reserved keywords that are never data fields.
// They are stripped when extracting meaningful output from an inner call
// response (for key-binding field selection and spread merging).
var protocolFieldKeys = map[string]bool{
	"ok":            true,
	"cancelled":     true,
	"error":         true,
	"custom_error":  true,
	"capture":       true,
	"cast":          true,
	"code":          true,
	"what":          true,
	"spore_error":   true,
	"node_error":    true,
	"cast_error":    true,
	"capture_error": true,
}

// ---- Pending state ----------------------------------------------------------

// pendingSlot is one outstanding inner call within a pending inline entry.
// The router keeps a map[internalHandle]*pendingSlot so it can find the right
// entry when a hub-internal response arrives.
type pendingSlot struct {
	internalHandle string          // handle value e.g. "~2BD4" (wire token: "~~2BD4")
	kind           message.ArgKind // ArgKindKeyBind or ArgKindSpread
	argIndex       int             // position in entry.held.Args at creation time
	resolvedArgs   []message.Arg  // filled in on successful inner response
	innerReceiver  string         // node ID the inner cast was routed to; used by PurgeNode
	resolved       bool
	entry          *pendingEntry
}

// pendingEntry tracks one outer cast held while its inline slots are resolved.
type pendingEntry struct {
	callerNodeID    string
	originalHandle  string
	originalCommand string
	held            message.ParsedArgs
	slots           []*pendingSlot
	shortCircuited  bool
}

// ---- Handle generation ------------------------------------------------------

// generateInlineHandle returns a hub-internal handle value in the form ~XXXX
// where XXXX is 4 random uppercase hex digits. On the wire this handle token
// appears as ~~XXXX (the outer ~ is the normal handle-token prefix).
//
// The leading ~ in the value means Capture.Parse strips the outer ~ and yields
// handle = "~XXXX" — exactly what is stored as the key in r.pending.
func generateInlineHandle() string {
	b := make([]byte, 2)
	_, _ = rand.Read(b)
	return fmt.Sprintf("~%04X", uint16(b[0])<<8|uint16(b[1]))
}

// ---- Outer cast routing -----------------------------------------------------

// inlineMsg is the subset of message.Message needed by routeInline.
// Both *message.Cast and *message.Spore satisfy this interface.
type inlineMsg interface {
	message.Message
	GetParsedArgs() (message.ParsedArgs, error)
}

// routeInline handles a Cast or Spore message that has inline call forms
// (key-binding or spread). It validates the forms, builds a pendingEntry,
// registers all slots, then fires every inner cast and returns immediately —
// the outer call is "held" in the pending entry until all slots resolve.
func (r *Router) routeInline(cast inlineMsg) error {
	p, err := cast.GetParsedArgs()
	if err != nil {
		r.SendError(cast, message.ErrorCodeMessageMalformed, "failed to parse inline args: "+err.Error())
		return err
	}

	// Spec §6.4: at most one spread inline per outer call.
	if p.SpreadCount() > 1 {
		r.SendError(cast, message.ErrorCodeMessageNotValid,
			"only one spread inline form is permitted per call")
		return errors.New("multiple spread forms in one outer call")
	}

	entry := &pendingEntry{
		callerNodeID:    cast.Cast(),
		originalHandle:  cast.Handle(),
		originalCommand: cast.Command(),
		held:            p,
	}

	// Register all slots under the lock BEFORE firing any inner cast, so
	// responses can never arrive before the slots are in r.pending.
	r.mu.Lock()
	for i, arg := range p.Args {
		if arg.Kind != message.ArgKindKeyBind && arg.Kind != message.ArgKindSpread {
			continue
		}
		handleVal := generateInlineHandle()
		slot := &pendingSlot{
			internalHandle: handleVal,
			kind:           arg.Kind,
			argIndex:       i,
			entry:          entry,
		}
		entry.slots = append(entry.slots, slot)
		r.pending[handleVal] = slot
	}
	r.mu.Unlock()

	// Fire all inner casts.
	for _, slot := range entry.slots {
		arg := p.Args[slot.argIndex]
		wireToken := "~" + slot.internalHandle // "~" + "~XXXX" = "~~XXXX"
		rawInner := arg.Inner + " " + wireToken

		innerMsg, parseErr := message.Parse(rawInner, entry.callerNodeID)
		if parseErr != nil {
			r.cleanupInlineEntry(entry)
			r.SendError(cast, message.ErrorCodeMessageMalformed,
				fmt.Sprintf("invalid inline call %q: %s", arg.Inner, parseErr.Error()))
			return parseErr
		}

		receiver, routeErr := r.routeInnerCast(innerMsg)
		if routeErr != nil {
			r.cleanupInlineEntry(entry)
			r.SendError(cast, message.ErrorCodeRouteNotFound,
				fmt.Sprintf("inner call %q routing failed: %s", innerMsg.Command(), routeErr.Error()))
			return routeErr
		}
		slot.innerReceiver = receiver
	}

	return nil
}

// routeInnerCast routes an inner cast created by the inline resolver.
//  - If the inner cast itself contains inline forms, recurse into routeInline
//    (this is how nested inlines — Phase 4.2 — work naturally).
//  - Otherwise, send directly to the target node. We deliberately do NOT add
//    the handle to r.replies; the response will be caught by the pending map
//    check at the top of Route instead.
//
// Returns the receiver node ID on success (empty string for nested inlines
// where no single direct receiver applies). Used by routeInline to populate
// slot.innerReceiver for PurgeNode.
func (r *Router) routeInnerCast(msg message.Message) (string, error) {
	// Nested inline detection — resolve inlines in the inner call first.
	if cast, ok := msg.(*message.Cast); ok && cast.HasInlines() {
		return "", r.routeInline(cast)
	}

	command := msg.Command()
	r.mu.Lock()
	defer r.mu.Unlock()

	nodeid, has := r.commands[command]
	if !has {
		return "", errors.New("command not registered: " + command)
	}
	node, err := r.hub.GetNode(nodeid)
	if err != nil {
		return "", err
	}
	return nodeid, node.Send(msg)
}

// ---- Inner response handling ------------------------------------------------

// handleInlineResponse is called when a response (capture/cancelled/error)
// arrives for a hub-internal handle registered in r.pending.
func (r *Router) handleInlineResponse(handle string, msg message.Message) error {
	r.mu.Lock()

	slot, ok := r.pending[handle]
	if !ok {
		// Race: already cleaned up by a concurrent short-circuit.
		r.mu.Unlock()
		return nil
	}
	entry := slot.entry

	if entry.shortCircuited {
		// A previous slot already short-circuited; discard this late response.
		delete(r.pending, handle)
		delete(r.replies, handle)
		r.mu.Unlock()
		return nil
	}

	// Remove from both maps (pending always; replies in case a re-routed cast
	// also registered it there through the normal Cast path).
	delete(r.pending, handle)
	delete(r.replies, handle)

	if msg.IsError() || msg.IsCancelled() {
		// Short-circuit: clean up all other outstanding slots and respond to
		// the original caller with the error or cancelled.
		entry.shortCircuited = true
		for _, s := range entry.slots {
			if !s.resolved {
				delete(r.pending, s.internalHandle)
				delete(r.replies, s.internalHandle)
			}
		}
		r.mu.Unlock()
		return r.shortCircuitEntry(entry, msg)
	}

	// Successful capture: resolve the slot's data fields.
	if err := r.resolveSlot(slot, msg); err != nil {
		entry.shortCircuited = true
		for _, s := range entry.slots {
			if !s.resolved {
				delete(r.pending, s.internalHandle)
				delete(r.replies, s.internalHandle)
			}
		}
		r.mu.Unlock()
		return r.sendInlineError(entry, message.ErrorCodeMessageNotValid, err.Error())
	}
	slot.resolved = true

	// Check whether all slots for this entry are now resolved.
	allDone := true
	for _, s := range entry.slots {
		if !s.resolved {
			allDone = false
			break
		}
	}
	r.mu.Unlock()

	if allDone {
		return r.completeInlineEntry(entry)
	}
	return nil
}

// ---- Slot resolution --------------------------------------------------------

// resolveSlot extracts the appropriate data field(s) from a successful inner
// response and stores them in slot.resolvedArgs.
func (r *Router) resolveSlot(slot *pendingSlot, msg message.Message) error {
	dataArgs := dataFieldsFrom(msg)
	switch slot.kind {
	case message.ArgKindKeyBind:
		return resolveKeyBind(slot, dataArgs)
	case message.ArgKindSpread:
		slot.resolvedArgs = dataArgs
		return nil
	}
	return errors.New("unknown slot kind")
}

// resolveKeyBind applies the spec §6.4 key-binding field selection rules:
//  1. Exactly one data field → use it regardless of its name, bound to the outer key.
//  2. Multiple data fields → find one whose name matches the outer key.
//  3. No match found → error (caller must handle this).
func resolveKeyBind(slot *pendingSlot, dataArgs []message.Arg) error {
	outerKey := slot.entry.held.Args[slot.argIndex].Key

	if len(dataArgs) == 0 {
		return fmt.Errorf("inner call returned no data fields for key binding %q", outerKey)
	}
	if len(dataArgs) == 1 {
		inner := dataArgs[0]
		raw := outerKey + "=" + inner.Value
		slot.resolvedArgs = []message.Arg{
			{Kind: message.ArgKindKeyValue, Key: outerKey, Value: inner.Value, Raw: raw},
		}
		return nil
	}
	// Multiple fields: look for a name match.
	for _, arg := range dataArgs {
		if arg.Key == outerKey {
			raw := outerKey + "=" + arg.Value
			slot.resolvedArgs = []message.Arg{
				{Kind: message.ArgKindKeyValue, Key: outerKey, Value: arg.Value, Raw: raw},
			}
			return nil
		}
	}
	return fmt.Errorf("inner call has multiple fields and none match key %q", outerKey)
}

// dataFieldsFrom tokenises a response message (ToString) and returns all
// argument tokens that are not protocol-reserved fields. The first token
// (~handle:subject) is always skipped.
func dataFieldsFrom(msg message.Message) []message.Arg {
	parts, err := message.Tokenize(msg.ToString())
	if err != nil || len(parts) < 2 {
		return nil
	}
	var out []message.Arg
	for _, tok := range parts[1:] {
		arg := message.ParseArgToken(tok)
		if arg.Kind == message.ArgKindHandle {
			continue // skip the handle token itself
		}
		if protocolFieldKeys[arg.Key] {
			continue // skip ok, capture, cast, code, what, flags like ok/cancelled/error...
		}
		out = append(out, arg)
	}
	return out
}

// ---- Completion and short-circuit -------------------------------------------

// completeInlineEntry applies all resolved slot values to the held outer call
// in descending argIndex order (so earlier positions remain stable), rebuilds
// the message string, and re-routes it as a normal cast.
func (r *Router) completeInlineEntry(entry *pendingEntry) error {
	type resolution struct {
		argIndex int
		args     []message.Arg
	}
	resolutions := make([]resolution, 0, len(entry.slots))
	for _, slot := range entry.slots {
		resolutions = append(resolutions, resolution{slot.argIndex, slot.resolvedArgs})
	}
	// Descending order keeps earlier arg indices stable as we splice.
	sort.Slice(resolutions, func(i, j int) bool {
		return resolutions[i].argIndex > resolutions[j].argIndex
	})
	for _, res := range resolutions {
		entry.held.ReplaceArg(res.argIndex, res.args...)
	}

	rawResolved := entry.held.Rebuild()
	slog.Debug("Inline resolution complete, re-routing", "raw", rawResolved, "caller", entry.callerNodeID)

	resolvedMsg, err := message.Parse(rawResolved, entry.callerNodeID)
	if err != nil {
		return r.sendInlineError(entry, message.ErrorCodeMessageMalformed,
			"failed to rebuild resolved message: "+err.Error())
	}
	return r.Route(resolvedMsg)
}

// shortCircuitEntry propagates an error or cancelled response from an inner
// call back to the original outer caller, abandoning the outer call.
func (r *Router) shortCircuitEntry(entry *pendingEntry, innerMsg message.Message) error {
	if innerMsg.IsCancelled() {
		return r.sendInlineCancelled(entry)
	}
	// Preserve the inner error's code and what if available.
	type coder interface {
		Code() message.ErrorCode
		What() string
	}
	code := message.ErrorCodeUnknownFailure
	what := "inner call failed"
	if c, ok := innerMsg.(coder); ok {
		code = c.Code()
		what = c.What()
	}
	return r.sendInlineError(entry, code, what)
}

// sendInlineError builds and delivers a hub-generated error to the original
// outer caller.
func (r *Router) sendInlineError(entry *pendingEntry, code message.ErrorCode, what string) error {
	raw := fmt.Sprintf(`~%s:%s error spore_error code=%s what="%s" capture=SPORE.hub`,
		entry.originalHandle, entry.originalCommand, code, what)
	errMsg, err := message.Parse(raw, "SPORE.hub")
	if err != nil {
		slog.Warn("Failed to build inline error message", "error", err)
		return err
	}
	callerNode, err := r.hub.GetNode(entry.callerNodeID)
	if err != nil {
		slog.Warn("Inline error: original caller unavailable", "caller", entry.callerNodeID)
		return err
	}
	return callerNode.Send(errMsg)
}

// sendInlineCancelled builds and delivers a cancelled response to the original
// outer caller when an inner call responded with cancelled.
func (r *Router) sendInlineCancelled(entry *pendingEntry) error {
	raw := fmt.Sprintf("~%s:%s cancelled capture=SPORE.hub",
		entry.originalHandle, entry.originalCommand)
	cancelledMsg, err := message.Parse(raw, "SPORE.hub")
	if err != nil {
		slog.Warn("Failed to build inline cancelled message", "error", err)
		return err
	}
	callerNode, err := r.hub.GetNode(entry.callerNodeID)
	if err != nil {
		slog.Warn("Inline cancelled: original caller unavailable", "caller", entry.callerNodeID)
		return err
	}
	return callerNode.Send(cancelledMsg)
}

// cleanupInlineEntry removes all slots for an entry from the pending map.
// Called when an error occurs during the setup phase before all inner casts
// have fired, to avoid leaving stale pending entries.
func (r *Router) cleanupInlineEntry(entry *pendingEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, slot := range entry.slots {
		delete(r.pending, slot.internalHandle)
	}
}
