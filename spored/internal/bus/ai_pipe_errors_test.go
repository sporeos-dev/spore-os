// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"fmt"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"strings"
	"testing"
)

func TestPipeIntermediateErrorReturnsOriginalResponse(t *testing.T) {
	b := New()
	b.Set(nil, nil, cancellationTestPermissions{}, nil)
	caller := &cancellationTestNode{id: "caller", received: make(chan iface.Message, 1)}
	target := &cancellationTestNode{
		id:       "target",
		manifest: &manifest.Manifest{Api: []manifest.Command{{Name: "first"}, {Name: "last"}}},
		received: make(chan iface.Message, 2),
	}
	b.Register(caller)
	b.Register(target)
	pipe, ok := message.Pipe("first ~original | last", caller.Id())
	if !ok {
		t.Fatal("invalid pipeline")
	}
	go b.Pipe(pipe, caller)
	request := receiveCancellationTestMessage(t, target.received)
	response, ok := message.Response(fmt.Sprintf("~%s:first error code=Failed.Node what=\"failed\" node_error", request.Handle()), target.Id())
	if !ok {
		t.Fatal("invalid error response")
	}
	if err := b.Response(response); err != nil {
		t.Fatalf("routing response: %v", err)
	}
	reply := receiveCancellationTestMessage(t, caller.received)
	if reply.Handle() != "original" || !reply.Flag("error") || reply.ArgIf("code", "") != "Failed.Node" ||
		reply.ArgIf("what", "") != "failed" || reply.Capture() != target.Id() {
		t.Fatalf("unexpected error response: %s", reply.Wire())
	}
	select {
	case msg := <-target.received:
		t.Fatalf("stage executed after error: %s", msg.Wire())
	default:
	}
}

func TestPipeIntermediateRoutingFailureReturnsOriginalHandle(t *testing.T) {
	b := New()
	b.Set(nil, nil, cancellationTestPermissions{}, nil)
	caller := &cancellationTestNode{id: "caller", received: make(chan iface.Message, 1)}
	b.Register(caller)
	pipe, ok := message.Pipe("missing ~original | last", caller.Id())
	if !ok {
		t.Fatal("invalid pipeline")
	}
	go b.Pipe(pipe, caller)
	reply := receiveCancellationTestMessage(t, caller.received)
	if reply.Handle() != "original" || !strings.HasPrefix(reply.Wire(), "~original:") ||
		!strings.Contains(reply.Wire(), "error") {
		t.Fatalf("unexpected routing error: %s", reply.Wire())
	}
}
