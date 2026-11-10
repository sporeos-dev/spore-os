package mycelium

import (
	"reflect"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	"spored/internal/manifest"
	"testing"
)

func TestWithdrawalPreservesOtherAbbreviationCandidates(t *testing.T) {
	event.Init()
	commands := newCommandMap()
	topics := newTopics()
	first := &manifest.Manifest{
		ID:     "clock",
		Api:    []manifest.Command{{Name: "tick"}},
		Topics: []manifest.Topic{{Name: "tick"}},
	}
	second := &manifest.Manifest{
		ID:     "timer",
		Api:    []manifest.Command{{Name: "tick"}},
		Topics: []manifest.Topic{{Name: "tick"}},
	}
	for _, m := range []*manifest.Manifest{first, second} {
		commands.OnManifestAnnouncement(manifest_event.New(m))
		topics.OnManifestAnnouncement(manifest_event.New(m))
	}
	topics.subscribers["timer.tick"] = []string{}
	if _, ok := commands.retrieveNode("tick"); ok {
		t.Error("ambiguous command unexpectedly resolved")
	}
	if _, ok := topics.retrieveSubs("tick", second.ID); ok {
		t.Error("ambiguous topic unexpectedly resolved")
	}

	commands.OnManifestWithdrawal(manifest_event.New(first))
	topics.OnManifestWithdrawal(manifest_event.New(first))
	if node, ok := commands.retrieveNode("tick"); !ok || node != "timer" {
		t.Errorf("command = %q, %t; want timer, true", node, ok)
	}
	if subs, ok := topics.retrieveSubs("tick", "timer"); !ok || len(subs) != 0 {
		t.Errorf("subscribers = %v, %t; want empty, true", subs, ok)
	}
}

func TestAbbreviationCollisionWithinManifest(t *testing.T) {
	event.Init()
	commands := newCommandMap()
	topics := newTopics()
	m := &manifest.Manifest{
		ID:     "node",
		Api:    []manifest.Command{{Name: "spawn.x"}, {Name: "kill.x"}},
		Topics: []manifest.Topic{{Name: "spawn.x"}, {Name: "kill.x"}},
	}

	commands.OnManifestAnnouncement(manifest_event.New(m))
	topics.OnManifestAnnouncement(manifest_event.New(m))
	topics.subscribers["node.spawn.x"] = []string{}
	if _, ok := commands.retrieveNode("x"); ok {
		t.Error("ambiguous command unexpectedly resolved")
	}
	if _, ok := topics.retrieveSubs("x", "node"); ok {
		t.Error("ambiguous topic unexpectedly resolved")
	}
	if node, ok := commands.retrieveNode("spawn.x"); !ok || node != "node" {
		t.Errorf("qualified command = %q, %t; want node, true", node, ok)
	}
	if _, ok := topics.retrieveSubs("spawn.x", "node"); !ok {
		t.Error("qualified topic did not resolve")
	}
	commands.OnManifestWithdrawal(manifest_event.New(m))
	topics.OnManifestWithdrawal(manifest_event.New(m))
	if len(commands.commands) != 0 || len(commands.abbreviations) != 0 ||
		len(topics.publishers) != 0 || len(topics.abbreviations) != 0 {
		t.Error("withdrawal left registrations behind")
	}
}

func TestWithdrawalDoesNotRemoveAnotherTopicOwner(t *testing.T) {
	topics := newTopics()
	owner := &manifest.Manifest{ID: "r.x", Topics: []manifest.Topic{{Name: "y"}}}
	other := &manifest.Manifest{ID: "r", Topics: []manifest.Topic{{Name: "x.y"}}}
	topics.OnManifestAnnouncement(manifest_event.New(owner))
	topics.OnManifestWithdrawal(manifest_event.New(other))

	if got, ok := topics.publishers["r.x.y"]; !ok || got != owner.ID {
		t.Errorf("publisher = %q, present = %t; want %q", got, ok, owner.ID)
	}
	if got := topics.abbreviations["y"]; !reflect.DeepEqual(got, []string{"r.x.y"}) {
		t.Errorf("abbreviation candidates = %v; want [r.x.y]", got)
	}
}

func TestTopicManifestEventSubscription(t *testing.T) {
	event.Init()
	topics := newTopics()
	event.Subscribe(topics)
	defer event.Unsubscribe(topics)
	m := &manifest.Manifest{ID: "node", Topics: []manifest.Topic{{Name: "tick"}}}

	event.Dispatch().ManifestAnnouncement.Notify(manifest_event.New(m))
	if got, ok := topics.publishers["node.tick"]; !ok || got != "node" {
		t.Errorf("publisher after announcement = %q, present = %t; want node", got, ok)
	}
	event.Dispatch().ManifestWithdrawal.Notify(manifest_event.New(m))
	if len(topics.publishers) != 0 || len(topics.abbreviations) != 0 {
		t.Errorf("withdrawal left entries: publishers=%v abbreviations=%v", topics.publishers, topics.abbreviations)
	}
}
