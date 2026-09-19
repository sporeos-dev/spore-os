package bus

import (
	"spored/internal/cparser"
	"testing"
	"time"
)

func TestStageTimeoutAbsent(t *testing.T) {
	pm := cparser.ParsedMessage{Args: map[string]string{}}
	d, ok := handleTimeout(&pm)
	if !ok {
		t.Fatalf("expected ok=true when timeout is absent")
	}
	if d != nil {
		t.Fatalf("expected nil override when timeout is absent, got %v", *d)
	}
}

func TestStageTimeoutFalse(t *testing.T) {
	pm := cparser.ParsedMessage{Args: map[string]string{"timeout": "false"}}
	d, ok := handleTimeout(&pm)
	if !ok {
		t.Fatalf("expected ok=true for timeout=false")
	}
	if d == nil || *d != 0 {
		t.Fatalf("expected override of 0 (no timeout) for timeout=false, got %v", d)
	}
	if _, present := pm.Args["timeout"]; present {
		t.Fatalf("expected timeout arg to be stripped from pm.Args")
	}
}

func TestStageTimeoutSeconds(t *testing.T) {
	pm := cparser.ParsedMessage{Args: map[string]string{"timeout": "30"}}
	d, ok := handleTimeout(&pm)
	if !ok {
		t.Fatalf("expected ok=true for timeout=30")
	}
	if d == nil || *d != 30*time.Second {
		t.Fatalf("expected override of 30s, got %v", d)
	}
	if _, present := pm.Args["timeout"]; present {
		t.Fatalf("expected timeout arg to be stripped from pm.Args")
	}
}

func TestStageTimeoutInvalid(t *testing.T) {
	pm := cparser.ParsedMessage{Args: map[string]string{"timeout": "soon"}}
	_, ok := handleTimeout(&pm)
	if ok {
		t.Fatalf("expected ok=false for a non-numeric, non-false timeout value")
	}
}

func TestStageTimeoutNegative(t *testing.T) {
	pm := cparser.ParsedMessage{Args: map[string]string{"timeout": "-5"}}
	_, ok := handleTimeout(&pm)
	if ok {
		t.Fatalf("expected ok=false for a negative timeout value")
	}
}
