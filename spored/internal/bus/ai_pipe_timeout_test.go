// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"fmt"
	"spored/internal/cparser"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	sporeerror "spored/internal/utilities/error"
	"strings"
	"testing"
	"time"
)

type pipeTimeoutNode struct {
	id       string
	manifest *manifest.Manifest
	received chan iface.Message
}

func (n *pipeTimeoutNode) Id() string                      { return n.id }
func (n *pipeTimeoutNode) IsConnected() bool               { return true }
func (n *pipeTimeoutNode) GetManifest() *manifest.Manifest { return n.manifest }
func (n *pipeTimeoutNode) Witness(iface.Message)           {}
func (n *pipeTimeoutNode) Receive(msg iface.Message) *sporeerror.Error {
	n.received <- msg
	return nil
}

type pipeTimeoutPermissions struct{}

func (pipeTimeoutPermissions) Can(iface.Message, string) (bool, *sporeerror.Error) {
	return true, nil
}

func newPipeTimeoutTestBus() (*Bus, *pipeTimeoutNode, *pipeTimeoutNode) {
	b := New()
	b.Set(nil, nil, pipeTimeoutPermissions{}, nil)
	b.pipe.pending.WithTimeout(30 * time.Millisecond)
	caller := &pipeTimeoutNode{id: "caller", received: make(chan iface.Message, 2)}
	target := &pipeTimeoutNode{
		id: "target",
		manifest: &manifest.Manifest{Api: []manifest.Command{
			{Name: "first"}, {Name: "second"}, {Name: "last"},
		}},
		received: make(chan iface.Message, 3),
	}
	b.Register(caller)
	b.Register(target)
	return b, caller, target
}

func startTimeoutTestPipe(t *testing.T, b *Bus, caller *pipeTimeoutNode, raw string) {
	t.Helper()
	msg, ok := message.Pipe(raw, caller.Id())
	if !ok {
		t.Fatalf("invalid test pipeline %q", raw)
	}
	go b.Pipe(msg, caller)
}

func receiveTimeoutTestMessage(t *testing.T, ch <-chan iface.Message) iface.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for test message")
		return nil
	}
}

func assertPipeTimeout(t *testing.T, msg iface.Message, handle, capability string) {
	t.Helper()
	parsed, ok := cparser.Validate(msg.Wire())
	if !ok {
		t.Fatalf("timeout wire is not a valid message: %q", msg.Wire())
	}
	if parsed.Type != cparser.TypeResponse || parsed.Handle != handle ||
		parsed.Command != capability || parsed.Args["code"] != "Timeout.Pipe" {
		t.Fatalf("unexpected timeout response: %q", msg.Wire())
	}
}

func TestPipeTimeoutReportsCallerHandle(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{name: "handle on first stage", raw: "first ~original | last"},
		{name: "client handle on last stage", raw: "first | last ~original"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, caller, target := newPipeTimeoutTestBus()
			startTimeoutTestPipe(t, b, caller, tt.raw)
			receiveTimeoutTestMessage(t, target.received)
			assertPipeTimeout(t, receiveTimeoutTestMessage(t, caller.received), "original", "first")
			select {
			case msg := <-target.received:
				t.Fatalf("unexpected later stage: %s", msg.Wire())
			default:
			}
		})
	}
}

func TestPipeTimeoutReportsLaterIntermediateStage(t *testing.T) {
	b, caller, target := newPipeTimeoutTestBus()
	startTimeoutTestPipe(t, b, caller, "first ~original | second | last")
	first := receiveTimeoutTestMessage(t, target.received)
	response, ok := message.Response(fmt.Sprintf("~%s:first ok", first.Handle()), target.Id())
	if !ok {
		t.Fatal("failed to construct first response")
	}
	if err := b.Response(response); err != nil {
		t.Fatalf("routing first response: %v", err)
	}
	second := receiveTimeoutTestMessage(t, target.received)
	if second.Capability() != "second" {
		t.Fatalf("routed %q, want second", second.Capability())
	}
	assertPipeTimeout(t, receiveTimeoutTestMessage(t, caller.received), "original", "second")
}

func TestPipeTimeoutFalseStillWaits(t *testing.T) {
	b, caller, target := newPipeTimeoutTestBus()
	startTimeoutTestPipe(t, b, caller, "first timeout=false | last ~original")
	first := receiveTimeoutTestMessage(t, target.received)
	if _, ok := first.Arg("timeout"); ok {
		t.Fatal("timeout argument was forwarded to the node")
	}
	select {
	case msg := <-caller.received:
		t.Fatalf("unexpected timeout: %s", msg.Wire())
	case <-time.After(100 * time.Millisecond):
	}
	response, ok := message.Response(fmt.Sprintf("~%s:first value=ready ok", first.Handle()), target.Id())
	if !ok {
		t.Fatal("failed to construct first response")
	}
	if err := b.Response(response); err != nil {
		t.Fatalf("routing first response: %v", err)
	}
	last := receiveTimeoutTestMessage(t, target.received)
	if last.Capability() != "last" || last.Handle() != "original" ||
		last.ArgIf("value", "") != "ready" {
		t.Fatalf("unexpected final stage: %s", last.Wire())
	}
}

func TestPipeTimeoutOverrideReportsCallerHandle(t *testing.T) {
	b, caller, target := newPipeTimeoutTestBus()
	startTimeoutTestPipe(t, b, caller, "first timeout=1 | last ~original")
	first := receiveTimeoutTestMessage(t, target.received)
	if strings.Contains(first.Wire(), "timeout=") {
		t.Fatalf("timeout argument was forwarded: %s", first.Wire())
	}
	select {
	case msg := <-caller.received:
		t.Fatalf("timeout fired before the override: %s", msg.Wire())
	case <-time.After(100 * time.Millisecond):
	}
	assertPipeTimeout(t, receiveTimeoutTestMessage(t, caller.received), "original", "first")
}
