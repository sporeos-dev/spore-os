// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package broadcaster

import (
	"errors"
	"log/slog"
	"spored/internal/interfaces"
	"spored/internal/message"
	"sync"
)

type Broadcaster struct {
	mu          sync.RWMutex
	topics      map[string]string   // topic name → publisher node ID
	subscribers map[string][]string // topic name → []subscriber node IDs
	hub         interfaces.Hub
}

func (b *Broadcaster) Open(hub interfaces.Hub) {
	b.hub = hub
	b.topics = make(map[string]string)
	b.subscribers = make(map[string][]string)
}

// ListTopics returns all registered topic names. If node is "n/a", all topics
// are returned. Otherwise only topics published by that node are returned.
func (b *Broadcaster) ListTopics(node string) []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	res := []string{}
	if node == "n/a" {
		for topic := range b.topics {
			res = append(res, topic)
		}
		return res
	}
	for topic, publisher := range b.topics {
		if publisher == node {
			res = append(res, topic)
		}
	}
	return res
}

// GetBroadcaster returns the publisher node ID for a topic.
func (b *Broadcaster) GetBroadcaster(topic string) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	publisher, has := b.topics[topic]
	if !has {
		return "", errors.New("topic not registered: " + topic)
	}
	return publisher, nil
}

// AddTopic registers a topic as published by the given node.
func (b *Broadcaster) AddTopic(topic string, publisherid string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.topics[topic] = publisherid
	if _, has := b.subscribers[topic]; !has {
		b.subscribers[topic] = []string{}
	}
}

// Subscribe registers a node as a subscriber to a topic.
func (b *Broadcaster) Subscribe(subscriberid string, topic string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[topic]
	for _, id := range subs {
		if id == subscriberid {
			return // already subscribed
		}
	}
	b.subscribers[topic] = append(subs, subscriberid)
}

// Unsubscribe removes a node from a topic's subscriber list.
func (b *Broadcaster) Unsubscribe(subscriberid string, topic string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[topic]
	for i, id := range subs {
		if id == subscriberid {
			b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
			return
		}
	}
}

// Publish fans out a publish message to all subscribers of the topic.
// It is fire-and-forget: unavailable or slow subscribers are skipped silently.
func (b *Broadcaster) Publish(msg message.Topic) error {
	b.mu.RLock()
	subs, has := b.subscribers[msg.TopicName()]
	if !has || len(subs) == 0 {
		b.mu.RUnlock()
		return nil
	}
	// Copy subscriber list before releasing the lock so sends don't hold it.
	subsCopy := make([]string, len(subs))
	copy(subsCopy, subs)
	b.mu.RUnlock()

	for _, subID := range subsCopy {
		node, err := b.hub.GetNode(subID)
		if err != nil {
			slog.Debug("Publish: subscriber unavailable", "topic", msg.TopicName(), "subscriber", subID)
			continue
		}
		msg.SetDestination(subID)
		if err := node.Send(msg); err != nil {
			slog.Warn("Publish: failed to deliver to subscriber", "topic", msg.TopicName(), "subscriber", subID, "error", err)
		}
	}
	return nil
}

// PurgeNode removes a node from all publisher and subscriber registrations.
// Called when a node disconnects.
func (b *Broadcaster) PurgeNode(nodeID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove as publisher — also drops the subscriber list for that topic.
	for topic, publisher := range b.topics {
		if publisher == nodeID {
			delete(b.topics, topic)
			delete(b.subscribers, topic)
		}
	}

	// Remove from all remaining subscriber lists.
	for topic, subs := range b.subscribers {
		for i, id := range subs {
			if id == nodeID {
				b.subscribers[topic] = append(subs[:i], subs[i+1:]...)
				break
			}
		}
	}
}
