// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package broadcaster

import (
	"errors"
	"spored/internal/interfaces"
	"spored/internal/manifest"
	"spored/internal/message"
	"testing"
)

// =============================================================================
// Stub implementations for testing
// =============================================================================

type stubNode struct {
	id   string
	sent []message.Message
}

func (n *stubNode) Send(msg message.Message) error {
	n.sent = append(n.sent, msg)
	return nil
}
func (n *stubNode) SendWitness(_ string)              {}
func (n *stubNode) GetManifest() *manifest.Manifest   { return &manifest.Manifest{ID: n.id} }
func (n *stubNode) IsConnected() bool                 { return true }
func (n *stubNode) GetPID() int                       { return 0 }

type stubHub struct {
	nodes map[string]*stubNode
}

func (h *stubHub) ListNodes() []string {
	ids := make([]string, 0, len(h.nodes))
	for id := range h.nodes {
		ids = append(ids, id)
	}
	return ids
}
func (h *stubHub) GetNode(id string) (interfaces.Node, error) {
	n, ok := h.nodes[id]
	if !ok {
		return nil, errors.New("node not found: " + id)
	}
	return n, nil
}
func (h *stubHub) AddNode(_ string) error                      { return nil }
func (h *stubHub) AddNodeWithManifest(_ *manifest.Manifest) error { return nil }
func (h *stubHub) RemoveNode(_ string) error                   { return nil }
func (h *stubHub) SpawnNode(_ string) error       { return nil }
func (h *stubHub) KillNode(_ string) error        { return nil }

func newBroadcasterWithHub() (*Broadcaster, *stubHub) {
	hub := &stubHub{nodes: make(map[string]*stubNode)}
	b := &Broadcaster{}
	b.Open(hub)
	return b, hub
}

func mustParsePublish(t *testing.T, raw, from string) message.Topic {
	t.Helper()
	msg, err := message.Parse(raw, from)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	pub, ok := msg.(message.Topic)
	if !ok {
		t.Fatal("expected message.Topic")
	}
	return pub
}

// =============================================================================
// AddTopic / ListTopics
// =============================================================================

func TestBroadcaster_AddAndListTopics(t *testing.T) {
	b, _ := newBroadcasterWithHub()
	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")
	b.AddTopic("SPORE.node.killed", "dev.sporeos.SPORE")

	all := b.ListTopics("n/a")
	if len(all) != 2 {
		t.Errorf("expected 2 topics, got %d", len(all))
	}

	filtered := b.ListTopics("dev.sporeos.SPORE")
	if len(filtered) != 2 {
		t.Errorf("expected 2 topics for publisher, got %d", len(filtered))
	}

	none := b.ListTopics("com.example.other")
	if len(none) != 0 {
		t.Errorf("expected 0 topics for unknown publisher, got %d", len(none))
	}
}

func TestBroadcaster_GetBroadcaster(t *testing.T) {
	b, _ := newBroadcasterWithHub()
	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")

	pub, err := b.GetBroadcaster("SPORE.node.installed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pub != "dev.sporeos.SPORE" {
		t.Errorf("expected publisher 'dev.sporeos.SPORE', got %q", pub)
	}

	_, err = b.GetBroadcaster("SPORE.node.unknown")
	if err == nil {
		t.Error("expected error for unknown topic")
	}
}

// =============================================================================
// Subscribe / Unsubscribe / Publish
// =============================================================================

func TestBroadcaster_PublishDeliveredToSubscribers(t *testing.T) {
	b, hub := newBroadcasterWithHub()

	sub1 := &stubNode{id: "com.example.sub1"}
	sub2 := &stubNode{id: "com.example.sub2"}
	hub.nodes["com.example.sub1"] = sub1
	hub.nodes["com.example.sub2"] = sub2

	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")
	b.Subscribe("com.example.sub1", "SPORE.node.installed")
	b.Subscribe("com.example.sub2", "SPORE.node.installed")

	pub := mustParsePublish(t, "publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err := b.Publish(pub); err != nil {
		t.Fatalf("Publish error: %v", err)
	}

	if len(sub1.sent) != 1 {
		t.Errorf("sub1: expected 1 message, got %d", len(sub1.sent))
	}
	if len(sub2.sent) != 1 {
		t.Errorf("sub2: expected 1 message, got %d", len(sub2.sent))
	}
}

func TestBroadcaster_PublishNoSubscribers(t *testing.T) {
	b, _ := newBroadcasterWithHub()
	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")

	pub := mustParsePublish(t, "publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	// Fire-and-forget — no subscribers is not an error.
	if err := b.Publish(pub); err != nil {
		t.Errorf("unexpected error with no subscribers: %v", err)
	}
}

func TestBroadcaster_Unsubscribe(t *testing.T) {
	b, hub := newBroadcasterWithHub()

	sub1 := &stubNode{id: "com.example.sub1"}
	hub.nodes["com.example.sub1"] = sub1

	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")
	b.Subscribe("com.example.sub1", "SPORE.node.installed")
	b.Unsubscribe("com.example.sub1", "SPORE.node.installed")

	pub := mustParsePublish(t, "publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	b.Publish(pub)

	if len(sub1.sent) != 0 {
		t.Errorf("expected no messages after unsubscribe, got %d", len(sub1.sent))
	}
}

func TestBroadcaster_SubscribeIdempotent(t *testing.T) {
	b, hub := newBroadcasterWithHub()

	sub1 := &stubNode{id: "com.example.sub1"}
	hub.nodes["com.example.sub1"] = sub1

	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")
	b.Subscribe("com.example.sub1", "SPORE.node.installed")
	b.Subscribe("com.example.sub1", "SPORE.node.installed") // duplicate

	pub := mustParsePublish(t, "publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	b.Publish(pub)

	if len(sub1.sent) != 1 {
		t.Errorf("expected 1 delivery despite duplicate subscribe, got %d", len(sub1.sent))
	}
}

func TestBroadcaster_PublishSkipsDisconnectedSubscriber(t *testing.T) {
	b, hub := newBroadcasterWithHub()
	// sub1 is registered but NOT in the hub's node map (simulates disconnect).
	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")
	b.Subscribe("com.example.gone", "SPORE.node.installed")

	sub2 := &stubNode{id: "com.example.sub2"}
	hub.nodes["com.example.sub2"] = sub2
	b.Subscribe("com.example.sub2", "SPORE.node.installed")

	pub := mustParsePublish(t, "publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err := b.Publish(pub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// sub2 should still receive it
	if len(sub2.sent) != 1 {
		t.Errorf("expected sub2 to receive message, got %d", len(sub2.sent))
	}
}

// =============================================================================
// PurgeNode
// =============================================================================

func TestBroadcaster_PurgeNode_RemovesSubscriber(t *testing.T) {
	b, hub := newBroadcasterWithHub()

	sub1 := &stubNode{id: "com.example.sub1"}
	hub.nodes["com.example.sub1"] = sub1

	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")
	b.Subscribe("com.example.sub1", "SPORE.node.installed")
	b.PurgeNode("com.example.sub1")

	pub := mustParsePublish(t, "publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	b.Publish(pub)

	if len(sub1.sent) != 0 {
		t.Errorf("expected no messages after PurgeNode, got %d", len(sub1.sent))
	}
}

func TestBroadcaster_PurgeNode_RemovesPublisherAndDropsTopic(t *testing.T) {
	b, _ := newBroadcasterWithHub()
	b.AddTopic("SPORE.node.installed", "dev.sporeos.SPORE")
	b.PurgeNode("dev.sporeos.SPORE")

	topics := b.ListTopics("n/a")
	if len(topics) != 0 {
		t.Errorf("expected topic to be removed after publisher purge, got %d topics", len(topics))
	}
}
