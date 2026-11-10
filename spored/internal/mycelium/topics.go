package mycelium

import (
	"slices"
	"spored/internal/base"
	manifest_event "spored/internal/event/manifest"
	"spored/internal/utilities/error"
	"spored/internal/witness"
	"strings"
	"sync"
)

type topics struct {
	*base.Base
	mu            sync.RWMutex
	subscribers   map[string][]string // topic -> subscribers
	publishers    map[string]string   // topic -> publisher
	abbreviations map[string][]string // multiple implies collision
}

func newTopics() *topics {
	return &topics{
		Base:          base.New().Set("mycelium", "topics"),
		subscribers:   make(map[string][]string),
		publishers:    make(map[string]string),
		abbreviations: make(map[string][]string),
	}
}

func (t *topics) retrieveSubs(topic string, origin string) ([]string, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	candidates := t.abbreviations[topic]
	if len(candidates) == 0 {
		witness.Error(error.Missing, error.Mycelium, "topic not found")
		return nil, false
	} else if len(candidates) > 1 {
		witness.Error(error.Collision, error.Mycelium, "topic collision")
		return nil, false
	}
	fqTopic := candidates[0]

	publisher, ok := t.publishers[fqTopic]
	if !ok {
		witness.Error(error.Missing, error.Mycelium, "publisher not found")
		return nil, false
	}

	if publisher != origin {
		witness.Error(error.NotPermitted, error.Mycelium, "incorrect publisher")
		return nil, false
	}

	subs, ok := t.subscribers[fqTopic]
	if !ok {
		witness.Error(error.Missing, error.Mycelium, "topic not found")
		return nil, false
	}
	return subs, ok
}

// manifest announcement
func (t *topics) OnManifestAnnouncement(ctx manifest_event.IManifestAnnouncementContext) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, topic := range ctx.Manifest().Topics {
		fqTopic := ctx.Manifest().ID + "." + topic.Name
		for _, abbr := range topicAbbreviations(fqTopic) {
			if !slices.Contains(t.abbreviations[abbr], fqTopic) {
				t.abbreviations[abbr] = append(t.abbreviations[abbr], fqTopic)
			}
		}
		t.publishers[fqTopic] = ctx.Manifest().ID
	}
}

// manifest withdrawal
func (t *topics) OnManifestWithdrawal(ctx manifest_event.IManifestWithdrawalContext) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, topic := range ctx.Manifest().Topics {
		fqTopic := ctx.Manifest().ID + "." + topic.Name
		publisher, ok := t.publishers[fqTopic]
		if !ok || publisher != ctx.Manifest().ID {
			continue
		}
		delete(t.publishers, fqTopic)
		for _, abbr := range topicAbbreviations(fqTopic) {
			remaining := slices.DeleteFunc(t.abbreviations[abbr], func(candidate string) bool {
				return candidate == fqTopic
			})
			if len(remaining) == 0 {
				delete(t.abbreviations, abbr)
			} else {
				t.abbreviations[abbr] = remaining
			}
		}
	}
}

//
//
// private
// helpers
//

func topicAbbreviations(fqTopic string) []string {
	parts := strings.Split(fqTopic, ".")
	abbreviations := make([]string, 0, len(parts))
	for i := len(parts) - 1; i >= 0; i-- {
		abbreviations = append(abbreviations, strings.Join(parts[i:], "."))
	}
	return abbreviations
}
