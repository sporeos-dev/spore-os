package permission

import (
	manifest_event "spored/internal/event/manifest"
	"spored/internal/manifest"
	"testing"
	"time"
)

func TestCanConsumesOnlyMatchingToken(t *testing.T) {
	a := New()
	now := time.Now()
	a.tokens = []temporaryGrant{
		{node: "caller", capability: "node.call", when: now, token: "first"},
		{node: "caller", capability: "node.call", when: now, token: "second"},
	}

	if a.Can("caller", "node.call", "node.call permission.token=wrong ~h") {
		t.Fatal("invalid token was accepted")
	}
	if a.Can("other", "node.call", "node.call permission.token=first ~h") {
		t.Fatal("token was accepted for another caller")
	}
	if a.Can("caller", "node.other", "node.other permission.token=first ~h") {
		t.Fatal("token was accepted for another capability")
	}
	if !a.Can("caller", "node.call", "node.call permission.token=first ~h") {
		t.Fatal("valid token was not accepted")
	}
	if a.Can("caller", "node.call", "node.call permission.token=first ~h") {
		t.Fatal("consumed token was accepted again")
	}
	if !a.Can("caller", "node.call", "node.call permission.token=second ~h") {
		t.Fatal("unrelated valid token was removed")
	}
}

func TestCanRejectsExpiredTokenWithoutRemovingValidToken(t *testing.T) {
	a := New()
	now := time.Now()
	a.tokens = []temporaryGrant{
		{node: "caller", capability: "node.call", when: now.Add(-30 * time.Second), token: "expired"},
		{node: "caller", capability: "node.call", when: now.Add(-15 * time.Second), token: "valid"},
	}

	if a.Can("caller", "node.call", "node.call permission.token=expired ~h") {
		t.Fatal("expired token was accepted")
	}
	if !a.Can("caller", "node.call", "node.call permission.token=valid ~h") {
		t.Fatal("valid token was removed along with the expired token")
	}
}

func TestCanDoesNotConsumeTokenWhenAlreadyGranted(t *testing.T) {
	a := New()
	a.grants["caller"] = []string{"node.call"}
	a.tokens = []temporaryGrant{
		{node: "caller", capability: "node.call", when: time.Now(), token: "valid"},
	}
	raw := "node.call permission.token=valid ~h"

	if !a.Can("caller", "node.call", raw) {
		t.Fatal("permanent grant was not accepted")
	}
	delete(a.grants, "caller")
	if !a.Can("caller", "node.call", raw) {
		t.Fatal("token was consumed despite the permanent grant")
	}
}

func TestCanConsumesTokenOnceConcurrently(t *testing.T) {
	a := New()
	a.tokens = []temporaryGrant{
		{node: "caller", capability: "node.call", when: time.Now(), token: "valid"},
	}

	const attempts = 64
	start := make(chan struct{})
	results := make(chan bool, attempts)
	for i := 0; i < attempts; i++ {
		go func() {
			<-start
			results <- a.Can("caller", "node.call", "node.call permission.token=valid ~h")
		}()
	}
	close(start)

	allowed := 0
	for i := 0; i < attempts; i++ {
		if <-results {
			allowed++
		}
	}

	if allowed != 1 {
		t.Errorf("token authorized %d requests, want 1", allowed)
	}
}

func TestCanUsesFullyQualifiedRiskNames(t *testing.T) {
	a := New()
	a.grants["caller"] = []string{"source.read", "other.read"}
	source := &manifest.Manifest{
		ID: "source", Api: []manifest.Command{{Name: "read", Risk: manifest.Protected}},
	}
	other := &manifest.Manifest{
		ID: "other", Api: []manifest.Command{{Name: "read", Risk: manifest.Protected}},
	}
	a.OnManifestAnnouncement(manifest_event.New(source))
	a.OnManifestAnnouncement(manifest_event.New(other))

	if a.Can("caller", "source.read", "read ~h") {
		t.Fatal("protected capability was allowed")
	}
	a.OnManifestWithdrawal(manifest_event.New(other))
	if a.Can("caller", "source.read", "read ~h") {
		t.Fatal("withdrawing another node removed protected status")
	}
	a.OnManifestWithdrawal(manifest_event.New(source))
	if !a.Can("caller", "source.read", "read ~h") {
		t.Fatal("grant was not usable after protected status was withdrawn")
	}
}
