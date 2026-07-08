// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package router

import (
	"errors"
	"fmt"
	"log/slog"
	"spored/internal/interfaces"
	"spored/internal/message"
	"strings"
	"sync"
	"time"
)

type Router struct {
	hub   interfaces.Hub
	spore interfaces.Spore

	mu          sync.RWMutex
	commands    map[string]string          // [clock.get_time] = dev.sporeos.clock
	replies     map[string]replyEntry      // [~handle] = {caster, receiver, command}
	pending     map[string]*pendingSlot    // [~~XXXX handle value] = inline slot
	hubRequests map[string]chan message.Message // [~XXXX] = reply channel for hub-initiated calls
}

// replyEntry tracks an in-flight call waiting for a response.
// It stores enough information for the router to:
//   - deliver the response to the right caster (caster field)
//   - find and purge stuck entries when a receiver disconnects (receiver field)
//   - build meaningful error messages on purge (command, handle fields)
type replyEntry struct {
	caster   string // node ID that sent the original cast
	receiver string // node ID that received the forwarded cast
	command  string // subject of the call (for error message construction)
	handle   string // redundant with map key, kept for convenience in PurgeNode
}

func (r *Router) Open(hub interfaces.Hub, spore interfaces.Spore) {
	r.hub = hub
	r.spore = spore
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands = make(map[string]string)
	r.pending = make(map[string]*pendingSlot)
	r.replies = make(map[string]replyEntry)
	r.hubRequests = make(map[string]chan message.Message)
}

func (r *Router) AddRoute(command string, nodeid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[command] = nodeid
}

// SendError delivers a hub-generated error message back to the original caster.
// It does not go through Route to avoid re-entrant locking or recursion.
func (r *Router) SendError(msg message.Message, code message.ErrorCode, what string) {
	hubErr := message.NewError(msg, code, what)
	nodeid := hubErr.Cast()
	node, err := r.hub.GetNode(nodeid)
	if err != nil {
		slog.Warn("Caster unavailable, could not deliver hub error", "caster", nodeid, "code", code)
		return
	}
	if err = node.Send(hubErr); err != nil {
		slog.Warn("Failed to deliver hub error to caster", "caster", nodeid, "code", code)
	}
}

func (r *Router) Route(msg message.Message) error {

	// === Pre-routing 1: response for a hub-internal (~~XXXX) inline handle ===
	// Hub-internal handles are keyed by their handle value (e.g. "~2BD4").
	// Check before the normal capture/error/cancelled dispatch so these
	// responses never reach the replies map lookup.
	if !msg.IsCast() && !msg.IsSpore() {
		handle := msg.Handle()
		r.mu.RLock()
		_, isPending := r.pending[handle]
		ch, isHubReq := r.hubRequests[handle]
		r.mu.RUnlock()
		if isPending {
			return r.handleInlineResponse(handle, msg)
		}
		if isHubReq {
			r.mu.Lock()
			delete(r.hubRequests, handle)
			r.mu.Unlock()
			ch <- msg
			return nil
		}
	}

	// === Pre-routing 2: cast/spore message containing inline call forms ===
	if msg.IsCast() {
		if cast, ok := msg.(*message.Cast); ok && cast.HasInlines() {
			return r.routeInline(cast)
		}
	} else if msg.IsSpore() {
		if spore, ok := msg.(*message.Spore); ok && spore.HasInlines() {
			return r.routeInline(spore)
		}
	}

	if msg.IsError() {

		if msg.IsCast() || msg.IsCapture() || msg.IsSpore() {
			nodeid := msg.Cast()
			node, err := r.hub.GetNode(nodeid)
			if err != nil {
				return err
			}
			if err = node.Send(msg); err != nil {
				return err
			}
			return nil
		}

		// NodeError arrives from the wire — need r.replies to find the original caster.
		handle := msg.Handle()

		r.mu.Lock()
		defer r.mu.Unlock()

		entry, has := r.replies[handle]
		if !has {
			return errors.New("error arrived with no registered caster for handle " + handle)
		}
		delete(r.replies, handle)

		node, err := r.hub.GetNode(entry.caster)
		if err != nil {
			return err
		}
		if err = node.Send(msg); err != nil {
			return err
		}
		return nil

	} else if msg.IsSpore() {

		response, err := r.spore.Command(msg)
		if err != nil {
			r.SendError(msg, message.ErrorCodeSporeFailure, err.Error())
			return err
		}
		if response == nil {
			r.SendError(msg, message.ErrorCodeSporeFailure, "spore command returned no response")
			return errors.New("spore command returned nil response")
		}
		nodeid := msg.Cast()
		node, err := r.hub.GetNode(nodeid)
		if err != nil {
			r.SendError(msg, message.ErrorCodeRouteNotConnected, "caster unavailable after spore command")
			return err
		}
		if err = node.Send(response); err != nil {
			r.SendError(msg, message.ErrorCodeConnectionFailure, "failed to deliver spore response")
			return err
		}

	} else if msg.IsCast() {

		cast := msg.Cast()
		command := msg.Command()
		handle := msg.Handle()

		r.mu.Lock()
		defer r.mu.Unlock()

		if _, has := r.commands[command]; !has {
			r.SendError(msg, message.ErrorCodeRouteNotFound, "command not registered: "+command)
			return errors.New("command not registered " + command)
		}
		if _, has := r.replies[handle]; has {
			r.SendError(msg, message.ErrorCodeHandleInUse, "handle already in use: "+handle)
			return errors.New("already waiting for handle " + handle)
		}

		nodeid := r.commands[command]
		r.replies[handle] = replyEntry{caster: cast, receiver: nodeid, command: command, handle: handle}
		node, err := r.hub.GetNode(nodeid)
		if err != nil {
			delete(r.replies, handle)
			r.SendError(msg, message.ErrorCodeRouteNotConnected, "receiver node unavailable: "+nodeid)
			return err
		}
		if !node.IsConnected() {
			delete(r.replies, handle)
			r.SendError(msg, message.ErrorCodeRouteNotConnected, "node is installed but not connected: "+nodeid)
			return errors.New("node not connected: " + nodeid)
		}
		if err = node.Send(msg); err != nil {
			delete(r.replies, handle)
			r.SendError(msg, message.ErrorCodeConnectionFailure, "failed to deliver cast to receiver")
			return err
		}

	} else if msg.IsCapture() {

		handle := msg.Handle()

		r.mu.Lock()
		defer r.mu.Unlock()

		entry, has := r.replies[handle]
		if !has {
			return errors.New("capture arrived with no registered caster for handle " + handle)
		}
		delete(r.replies, handle)

		node, err := r.hub.GetNode(entry.caster)
		if err != nil {
			return err
		}
		if err = node.Send(msg); err != nil {
			return err
		}
		return nil

	} else if msg.IsCancelled() {

		// Cancelled responses are routed back to the original caster exactly
		// like captures — look up the handle in replies, deliver, and clean up.
		handle := msg.Handle()

		r.mu.Lock()
		defer r.mu.Unlock()

		entry, has := r.replies[handle]
		if !has {
			return errors.New("cancelled arrived with no registered caster for handle " + handle)
		}
		delete(r.replies, handle)

		node, err := r.hub.GetNode(entry.caster)
		if err != nil {
			return err
		}
		if err = node.Send(msg); err != nil {
			return err
		}
		return nil

	} else {
		return errors.New("unhandled message-type")
	}

	return nil
}

func (r *Router) GetRoute(command string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	node, has := r.commands[command]
	if !has {
		return "", errors.New("route does not exist")
	}
	return node, nil
}

func (r *Router) ListCommands(node string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]string, 0, len(r.commands))

	if node == "n/a" {

		for commandid := range r.commands {
			res = append(res, commandid)
		}

	} else {

		for commandid := range r.commands {
			nodeid, _ := r.commands[commandid]
			if node != nodeid {
				continue
			}

			res = append(res, commandid)
		}
	}
	
	return res
}

// RemoveRoutes removes all commands registered for a node from the routing
// table. Called on uninstall (hub.RemoveNode) so that future calls to those
// subjects return RouteNotFound. This is distinct from PurgeNode, which only
// cleans up in-flight handles and is called on disconnect.
func (r *Router) RemoveRoutes(nodeID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for cmd, owner := range r.commands {
		if owner == nodeID {
			delete(r.commands, cmd)
		}
	}
}

// PurgeNode is called when a node disconnects. It finds all in-flight handles
// associated with the disconnected node and cleans up:
//
//   - Receiver side: the handle was waiting for a response from this node.
//     The caster is still alive and will never hear back, so we deliver a
//     RouteNotConnected error to it and free the handle immediately.
//
//   - Caster side: the handle was waiting to deliver a response to this node.
//     The caster is gone, so we silently discard — the receiver may still
//     respond but the response will fail gracefully when GetNode returns an
//     error (and the handle will be cleaned up at that point).
//
//   - Inline pending slots (receiver): the inner call was routed to this node;
//     the whole outer expression is short-circuited with an error to the
//     original outer caller.
//
//   - Inline pending slots (caster): the outer caster is gone; clean up the
//     slot map silently.
func (r *Router) PurgeNode(nodeID string) {
	r.mu.Lock()

	// NOTE: commands are intentionally NOT removed here. Routes persist while a
	// node is installed so that callers receive RouteNotConnected (not
	// RouteNotFound) when a node disconnects and reconnects. Use RemoveRoutes
	// to delete commands (called on uninstall via hub.RemoveNode).

	// Collect reply entries to act on.
	type pendingReply struct {
		handle   string
		command  string
		caster   string
		isReceiverGone bool // false = caster is the one gone
	}
	var toActOn []pendingReply
	for h, e := range r.replies {
		if e.receiver == nodeID {
			toActOn = append(toActOn, pendingReply{h, e.command, e.caster, true})
			delete(r.replies, h)
		} else if e.caster == nodeID {
			toActOn = append(toActOn, pendingReply{h, e.command, e.caster, false})
			delete(r.replies, h)
		}
	}

	// Collect inline pending entries to act on.
	type pendingInline struct {
		entry        *pendingEntry
		receiverGone bool
	}
	seen := make(map[*pendingEntry]bool)
	var inlineToActOn []pendingInline
	for h, slot := range r.pending {
		entry := slot.entry
		if slot.innerReceiver == nodeID && !entry.shortCircuited {
			if !seen[entry] {
				seen[entry] = true
				inlineToActOn = append(inlineToActOn, pendingInline{entry, true})
			}
		} else if entry.callerNodeID == nodeID {
			// Caster is gone — clean up all slots for this entry silently.
			if !seen[entry] {
				seen[entry] = true
				inlineToActOn = append(inlineToActOn, pendingInline{entry, false})
			}
		}
		_ = h
	}
	for _, pi := range inlineToActOn {
		pi.entry.shortCircuited = true
		for _, s := range pi.entry.slots {
			delete(r.pending, s.internalHandle)
		}
	}

	r.mu.Unlock()

	// Send errors to callers whose receiver disconnected (outside the lock).
	for _, p := range toActOn {
		if !p.isReceiverGone {
			continue // caster is gone — nobody to notify
		}
		r.sendPurgeError(p.handle, p.command, p.caster, nodeID)
	}
	for _, pi := range inlineToActOn {
		if !pi.receiverGone {
			continue
		}
		r.sendPurgeError(pi.entry.originalHandle, pi.entry.originalCommand, pi.entry.callerNodeID, nodeID)
	}
}

// sendPurgeError delivers a RouteNotConnected error to the named caster for a
// handle that was stuck because its receiver disconnected. Called outside the
// router lock.
func (r *Router) sendPurgeError(handle, command, casterNodeID, disconnectedNodeID string) {
	raw := fmt.Sprintf(`~%s:%s error spore_error code=%s what="receiver disconnected: %s"`,
		handle, command, message.ErrorCodeRouteNotConnected, disconnectedNodeID)
	errMsg, err := message.Parse(raw, "SPORE.hub")
	if err != nil {
		slog.Warn("PurgeNode: failed to build error", "handle", handle, "error", err)
		return
	}
	casterNode, err := r.hub.GetNode(casterNodeID)
	if err != nil {
		slog.Warn("PurgeNode: caster unavailable", "caster", casterNodeID, "handle", handle)
		return
	}
	if err = casterNode.Send(errMsg); err != nil {
		slog.Warn("PurgeNode: failed to deliver error to caster", "caster", casterNodeID, "handle", handle)
	}
}

const hubRequestTimeout = 10 * time.Second

// hubNodeID is the node ID the hub uses as the cast= origin for internal
// requests. Matches the id field in spored.manifest.spore.yaml.
const hubNodeID = "dev.sporeos.SPORE"

// RequestNode sends command to the named node and blocks until the reply
// arrives or hubRequestTimeout elapses. args values that contain spaces are
// automatically quoted. Returns an error if the node is not connected, the
// send fails, the node returns an error reply, or the call times out.
func (r *Router) RequestNode(nodeID string, command string, args map[string]string) (message.Message, error) {
	node, err := r.hub.GetNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("router: RequestNode %s: node not found: %w", command, err)
	}
	if !node.IsConnected() {
		return nil, fmt.Errorf("router: RequestNode %s: %s is not connected", command, nodeID)
	}

	handle := generateInlineHandle() // "~XXXX"
	wireToken := "~" + handle        // "~~XXXX" on the wire

	// Build raw cast string.
	var sb strings.Builder
	sb.WriteString(command)
	sb.WriteString(" ")
	sb.WriteString(wireToken)
	for k, v := range args {
		sb.WriteString(" ")
		sb.WriteString(k)
		sb.WriteString("=")
		if strings.ContainsAny(v, " \t\"") {
			sb.WriteString(`"`)
			sb.WriteString(strings.ReplaceAll(v, `"`, `\"`))
			sb.WriteString(`"`)
		} else {
			sb.WriteString(v)
		}
	}

	msg, err := message.Parse(sb.String(), hubNodeID)
	if err != nil {
		return nil, fmt.Errorf("router: RequestNode %s: build message: %w", command, err)
	}

	ch := make(chan message.Message, 1)
	r.mu.Lock()
	r.hubRequests[handle] = ch
	r.mu.Unlock()

	if err := node.Send(msg); err != nil {
		r.mu.Lock()
		delete(r.hubRequests, handle)
		r.mu.Unlock()
		return nil, fmt.Errorf("router: RequestNode %s: send: %w", command, err)
	}

	select {
	case reply := <-ch:
		if reply.IsError() {
			return nil, fmt.Errorf("router: RequestNode %s: node error: %s", command, reply.ToString())
		}
		return reply, nil
	case <-time.After(hubRequestTimeout):
		r.mu.Lock()
		delete(r.hubRequests, handle)
		r.mu.Unlock()
		return nil, fmt.Errorf("router: RequestNode %s to %s: timed out after %s", command, nodeID, hubRequestTimeout)
	}
}