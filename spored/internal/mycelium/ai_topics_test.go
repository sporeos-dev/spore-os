package mycelium

import (
	"spored/internal/event"
	"testing"
)

func TestRetrieveSubsReportsMissingTopicEntry(t *testing.T) {
	event.Init()

	topics := newTopics()
	topics.abbreviations["some.topic"] = []string{"some.topic"}
	topics.publishers["some.topic"] = "origin"

	subs, ok := topics.retrieveSubs("some.topic", "origin")
	if ok || subs != nil {
		t.Errorf("retrieveSubs(%q) = %v, %t; want nil, false", "some.topic", subs, ok)
	}
}

func TestRetrieveSubsAllowsEmptySubscriberList(t *testing.T) {
	topics := newTopics()
	topics.abbreviations["some.topic"] = []string{"some.topic"}
	topics.publishers["some.topic"] = "origin"
	topics.subscribers["some.topic"] = []string{}

	subs, ok := topics.retrieveSubs("some.topic", "origin")
	if !ok || len(subs) != 0 {
		t.Errorf("retrieveSubs(%q) = %v, %t; want an empty list, true", "some.topic", subs, ok)
	}
}

func TestRetrieveSubsChecksPublisher(t *testing.T) {
	event.Init()
	topics := newTopics()
	topics.abbreviations["tick"] = []string{"clock.tick"}
	topics.subscribers["clock.tick"] = []string{"listener"}

	if _, ok := topics.retrieveSubs("tick", "clock"); ok {
		t.Error("topic without a publisher unexpectedly resolved")
	}
	topics.publishers["clock.tick"] = "clock"
	if _, ok := topics.retrieveSubs("tick", "other"); ok {
		t.Error("unauthorized publisher unexpectedly resolved")
	}
	if subs, ok := topics.retrieveSubs("tick", "clock"); !ok || len(subs) != 1 || subs[0] != "listener" {
		t.Errorf("subscribers = %v, %t; want [listener], true", subs, ok)
	}
}
