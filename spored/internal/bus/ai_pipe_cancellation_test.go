// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"fmt"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	sporeerror "spored/internal/utilities/error"
	"strings"
	"testing"
	"time"
)

type cancellationTestNode struct {
	id       string
	manifest *manifest.Manifest
	received chan iface.Message
}

func (n *cancellationTestNode) Id() string                      { return n.id }
func (n *cancellationTestNode) IsConnected() bool               { return true }
func (n *cancellationTestNode) GetManifest() *manifest.Manifest { return n.manifest }
func (n *cancellationTestNode) Witness(iface.Message)           {}
func (n *cancellationTestNode) Receive(msg iface.Message) *sporeerror.Error {
	n.received <- msg
	return nil
}

type cancellationTestPermissions struct{}

func (cancellationTestPermissions) Can(iface.Message, string) (bool, *sporeerror.Error) {
	return true, nil
}

func receiveCancellationTestMessage(t *testing.T, ch <-chan iface.Message) iface.Message {
	t.Helper()
	select {
	case msg := <-ch:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message")
		return nil
	}
}

func TestPipeCancellationReturnsOriginalHandle(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		cancelStage int
	}{
		{"first stage, handle first", "first ~original | last", 0},
		{"first stage, handle last", "first | last ~original", 0},
		{"middle stage, handle first", "first ~original | second | last", 1},
		{"middle stage, handle last", "first | second | last ~original", 1},
		{"last stage, handle first", "first ~original | last", 1},
		{"last stage, handle last", "first | last ~original", 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := New()
			b.Set(nil, nil, cancellationTestPermissions{}, nil)
			caller := &cancellationTestNode{id: "caller", received: make(chan iface.Message, 2)}
			target := &cancellationTestNode{
				id:       "target",
				manifest: &manifest.Manifest{Api: []manifest.Command{{Name: "first"}, {Name: "second"}, {Name: "last"}}},
				received: make(chan iface.Message, 3),
			}
			b.Register(caller)
			b.Register(target)
			pipe, ok := message.Pipe(tt.raw, caller.Id())
			if !ok {
				t.Fatal("invalid pipeline")
			}
			done := make(chan struct{})
			go func() {
				b.Pipe(pipe, caller)
				close(done)
			}()

			for stage := 0; stage <= tt.cancelStage; stage++ {
				request := receiveCancellationTestMessage(t, target.received)
				if request.Flag("ok") || request.Flag("cancelled") {
					t.Fatalf("protocol flag leaked to request: %s", request.Wire())
				}
				flags := "ok value=ready"
				if stage == tt.cancelStage {
					flags = "ok cancelled reason=\"user dismissed\""
				}
				response, ok := message.Response(fmt.Sprintf("~%s:%s %s", request.Handle(), request.Capability(), flags), target.Id())
				if !ok {
					t.Fatal("invalid response")
				}
				if err := b.Response(response); err != nil {
					t.Fatalf("routing response: %v", err)
				}
			}

			reply := receiveCancellationTestMessage(t, caller.received)
			if reply.Handle() != "original" || !reply.Flag("ok") || !reply.Flag("cancelled") ||
				reply.Capture() != target.Id() || reply.ArgIf("reason", "") != "user dismissed" {
				t.Fatalf("unexpected cancellation: %s", reply.Wire())
			}
			if !strings.HasPrefix(reply.Wire(), "~original:") || !strings.Contains(reply.Wire(), "capture=target") {
				t.Fatalf("incorrect cancellation wire: %s", reply.Wire())
			}
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("pipeline did not stop after cancellation")
			}
			select {
			case msg := <-target.received:
				t.Fatalf("stage executed after cancellation: %s", msg.Wire())
			default:
			}
		})
	}
}
