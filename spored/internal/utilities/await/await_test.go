package await

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"testing"
	"time"
)

func TestWaitForTimeoutOverrideShorterThanDefault(t *testing.T) {
	p := New(error.Pipe).WithTimeout(time.Second)
	ch := p.Await("h1")

	override := 10 * time.Millisecond
	_, err := p.WaitForTimeout("h1", ch, &override)
	if err == nil {
		t.Fatalf("expected a timeout error, got none")
	}
}

func TestWaitForTimeoutOverrideDisabled(t *testing.T) {
	p := New(error.Pipe).WithTimeout(10 * time.Millisecond)
	ch := p.Await("h1")

	go func() {
		time.Sleep(30 * time.Millisecond)
		p.Receive(message.Signal("h1"))
	}()

	override := time.Duration(0)
	_, err := p.WaitForTimeout("h1", ch, &override)
	if err != nil {
		t.Fatalf("expected no timeout error with disabled override, got %v", err)
	}
}

func TestWaitForTimeoutNilOverrideUsesDefault(t *testing.T) {
	p := New(error.Pipe).WithTimeout(10 * time.Millisecond)
	ch := p.Await("h1")

	_, err := p.WaitForTimeout("h1", ch, nil)
	if err == nil {
		t.Fatalf("expected the default timeout to apply and produce an error")
	}
}
