// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"spored/internal/base"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/registry"
	sporeerror "spored/internal/utilities/error"
	"strings"
	"testing"
)

func TestReturnRoutingErrorReportsDeliveryFailure(t *testing.T) {
	subscriber := &writerWitnessSubscriber{
		Base:   base.New(),
		events: make(chan witness_event.IWitnessContext, 1),
	}
	event.Subscribe(subscriber)
	defer event.Unsubscribe(subscriber)

	node := newNode(&registry.Element{ID: "test.shell"}, &manifest.Manifest{})
	request, ok := message.Request("example.command ~request", node.Id())
	if !ok {
		t.Fatal("invalid request")
	}
	node.returnRoutingError(sporeerror.MissingArg("node", sporeerror.Spore), request)

	select {
	case ctx := <-subscriber.events:
		if ctx.Type() != witness_event.WTypeError ||
			!strings.Contains(ctx.Message(), "failed to return routing error") ||
			!strings.Contains(ctx.Message(), "node not connected") {
			t.Errorf("unexpected witness event: %q", ctx.Message())
		}
	default:
		t.Fatal("expected witness event for failed error delivery")
	}
}
