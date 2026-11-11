// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package await

import (
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"spored/internal/utilities/error"
	"testing"
	"time"
)

func TestWaitForTimeoutOverrideShorterThanDefault(t *testing.T) {
	event.Init()
	p := New(error.Pipe).WithTimeout(time.Second)
	ch := p.Await("h1")
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		timer := time.NewTimer(200 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
			p.Receive(message_event.Response("~h1:test ok", "test", "h1", "caller"))
		case <-stop:
		}
	}()

	override := 10 * time.Millisecond
	response, ok := p.WaitForTimeout("h1", ch, &override)
	close(stop)
	<-done
	if ok || response != nil {
		t.Fatalf("expected a timeout, got response %v and success %v", response, ok)
	}
	if p.Has("h1") {
		t.Fatal("timed-out handle is still pending")
	}
}

func TestWaitForTimeoutOverrideDisabled(t *testing.T) {
	event.Init()
	p := New(error.Pipe).WithTimeout(10 * time.Millisecond)
	ch := p.Await("h1")
	want := message_event.Response("~h1:test ok", "test", "h1", "caller")
	done := make(chan struct{})

	go func() {
		defer close(done)
		time.Sleep(30 * time.Millisecond)
		p.Receive(want)
	}()

	override := time.Duration(0)
	response, ok := p.WaitForTimeout("h1", ch, &override)
	<-done
	if !ok || response != want {
		t.Fatalf("expected the response with disabled timeout, got %v and success %v", response, ok)
	}
	if p.Has("h1") {
		t.Fatal("received handle is still pending")
	}
}

func TestWaitForTimeoutNilOverrideUsesDefault(t *testing.T) {
	event.Init()
	p := New(error.Pipe).WithTimeout(10 * time.Millisecond)
	ch := p.Await("h1")

	response, ok := p.WaitForTimeout("h1", ch, nil)
	if ok || response != nil {
		t.Fatalf("expected the default timeout, got response %v and success %v", response, ok)
	}
	if p.Has("h1") {
		t.Fatal("timed-out handle is still pending")
	}
}
