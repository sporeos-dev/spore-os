// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package message

import (
	"testing"
)

// =============================================================================
// Publish message type
// Wire format: publish <topic> [key=value ...] [flag ...]
// =============================================================================

func TestPublish_ParseBasic(t *testing.T) {
	msg, err := Parse("publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsPublish() {
		t.Error("expected IsPublish() to be true")
	}
	if msg.IsCast() || msg.IsCapture() || msg.IsSpore() || msg.IsError() || msg.IsCancelled() {
		t.Error("wrong type flags set on Publish")
	}
}

func TestPublish_TopicName(t *testing.T) {
	msg, err := Parse("publish SPORE.node.spawned node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	pub, ok := msg.(*Publish)
	if !ok {
		t.Fatal("expected *Publish concrete type")
	}
	if pub.TopicName() != "SPORE.node.spawned" {
		t.Errorf("expected topic 'SPORE.node.spawned', got %q", pub.TopicName())
	}
}

func TestPublish_SourceIsPublisher(t *testing.T) {
	msg, err := Parse("publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Source() != "dev.sporeos.SPORE" {
		t.Errorf("expected source 'dev.sporeos.SPORE', got %q", msg.Source())
	}
	if msg.Cast() != "dev.sporeos.SPORE" {
		t.Errorf("expected cast 'dev.sporeos.SPORE', got %q", msg.Cast())
	}
}

func TestPublish_NoHandle(t *testing.T) {
	msg, err := Parse("publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Handle() != "" {
		t.Errorf("expected empty handle, got %q", msg.Handle())
	}
}

func TestPublish_SetDestination(t *testing.T) {
	msg, err := Parse("publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	msg.SetDestination("com.example.subscriber")
	if msg.Destination() != "com.example.subscriber" {
		t.Errorf("expected destination 'com.example.subscriber', got %q", msg.Destination())
	}
}

func TestPublish_ToString(t *testing.T) {
	raw := "publish SPORE.node.installed node=dev.sporeos.clock"
	msg, err := Parse(raw, "dev.sporeos.SPORE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ToString() != raw {
		t.Errorf("expected ToString() %q, got %q", raw, msg.ToString())
	}
}

func TestPublish_MissingTopicErrors(t *testing.T) {
	_, err := Parse("publish ", "dev.sporeos.SPORE")
	if err == nil {
		t.Error("expected error for publish with no topic name")
	}
}

func TestPublish_ImplementsTopicInterface(t *testing.T) {
	msg, err := Parse("publish SPORE.node.installed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// *Publish must satisfy the Topic interface (which embeds Message).
	var _ Topic = msg.(*Publish)
}

func TestParse_DispatchesPublishForPublishPrefix(t *testing.T) {
	msg, err := Parse("publish SPORE.node.killed node=dev.sporeos.clock", "dev.sporeos.SPORE")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !msg.IsPublish() {
		t.Error("expected Publish message for 'publish ' prefix")
	}
}
