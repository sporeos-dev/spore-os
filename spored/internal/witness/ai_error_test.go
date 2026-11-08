// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package witness

import (
	"bytes"
	"log/slog"
	"spored/internal/base"
	"spored/internal/event"
	witness_event "spored/internal/event/witness"
	sporeerror "spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"strings"
	"testing"
)

type errorSubscriber struct {
	*base.Base
	events chan witness_event.IWitnessContext
}

func (s *errorSubscriber) OnWitness(ctx witness_event.IWitnessContext) {
	s.events <- ctx
}

func TestErrorLogsAndEmitsWitnessEvent(t *testing.T) {
	event.Init()
	subscriber := &errorSubscriber{
		Base:   base.New(),
		events: make(chan witness_event.IWitnessContext, 1),
	}
	event.Subscribe(subscriber)
	defer event.Unsubscribe(subscriber)

	var log bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&log, nil)))
	defer slog.SetDefault(previous)

	Error(sporeerror.InitializationFailure, sporeerror.Hub, "listen failed", out.Pair("error", "permission denied"))

	select {
	case ctx := <-subscriber.events:
		if ctx.Type() != witness_event.WTypeError {
			t.Errorf("event type = %q, want %q", ctx.Type(), witness_event.WTypeError)
		}
		for _, want := range []string{"InitializationFailure.Hub", "listen failed", "permission denied"} {
			if !strings.Contains(ctx.Message(), want) || !strings.Contains(log.String(), want) {
				t.Errorf("event and log must contain %q; event=%q log=%q", want, ctx.Message(), log.String())
			}
		}
	default:
		t.Fatal("expected witness error event")
	}
}

func TestErrorLogsWithoutWitnessSubscribers(t *testing.T) {
	event.Init()
	var log bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&log, nil)))
	defer slog.SetDefault(previous)

	Error(sporeerror.ConnectionFailure, sporeerror.Hub, "accept failed")

	if !strings.Contains(log.String(), "accept failed") {
		t.Fatalf("error was not logged: %q", log.String())
	}
}
