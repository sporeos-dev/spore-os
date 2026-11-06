// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"fmt"
	"spored/internal/cparser"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"testing"
)

func TestPipeRoutingErrorIncludesHubCapture(t *testing.T) {
	for _, tt := range []struct {
		name          string
		raw           string
		firstSucceeds bool
	}{
		{name: "intermediate stage", raw: "missing ~original | last"},
		{name: "final stage", raw: "first ~original | missing", firstSucceeds: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := New()
			b.Set(nil, nil, cancellationTestPermissions{}, nil)
			caller := &cancellationTestNode{id: "caller", received: make(chan iface.Message, 1)}
			target := &cancellationTestNode{
				id:       "target",
				manifest: &manifest.Manifest{Api: []manifest.Command{{Name: "first"}}},
				received: make(chan iface.Message, 1),
			}
			b.Register(caller)
			b.Register(target)
			pipe, ok := message.Pipe(tt.raw, caller.Id())
			if !ok {
				t.Fatal("invalid pipeline")
			}
			go b.Pipe(pipe, caller)
			if tt.firstSucceeds {
				request := receiveCancellationTestMessage(t, target.received)
				response, ok := message.Response(fmt.Sprintf("~%s:first ok", request.Handle()), target.Id())
				if !ok {
					t.Fatal("invalid response")
				}
				if err := b.Response(response); err != nil {
					t.Fatalf("routing response: %v", err)
				}
			}
			reply := receiveCancellationTestMessage(t, caller.received)
			parsed, ok := cparser.Parse(reply.Wire())
			if !ok {
				t.Fatalf("invalid error response: %s", reply.Wire())
			}
			if parsed.Handle != "original" || parsed.Args["capture"] != "dev.sporeos.SPORE" {
				t.Fatalf("unexpected error response: %s", reply.Wire())
			}
		})
	}
}
