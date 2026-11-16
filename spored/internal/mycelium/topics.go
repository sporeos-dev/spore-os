package mycelium

import (
	"fmt"
	"slices"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	message_event "spored/internal/event/message"
	node_event "spored/internal/event/node"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"strings"
	"sync"
)

var _ message_event.IRequestHandler = (*topics)(nil)
var _ node_event.INodeDisconnectionHandler = (*topics)(nil)

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
	return slices.Clone(subs), ok
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

// request
func (t *topics) OnRequest(ctx message_event.IRequestContext) {
	
	command := ctx.Command()
	message := ctx.Message()
	origin := ctx.Origin()
	handle := ctx.Handle()

	go func() {
		respondError := func(code error.Code, what string) {
			response := message_event.ResponseError(code, error.Persistence, what, handle, command, origin)
			event.Dispatch().Message.Notify(message_event.Raw(response.Message(), "dev.sporeos.SPORE"))
		}
		witnessError := func(code error.Code, what string, outs ...out.IOut) {
			witness.Error(code, error.Mycelium, what, outs...)
		}

		pm, ok := cparser.ParseAndValidate(message)
		if !ok {
			// witnessed in ParseAndValidate
			respondError(error.Malformed, "bad message")
			return
		}

		if command == "subscribe" || strings.HasSuffix(command, ".subscribe") {

			topic, ok := pm.Args["topic"]
			if !ok {
				witnessError(error.Missing, "mssing arg", out.Pair("arg", "topic"))
				respondError(error.Missing, "missing 'topic'")
				return
			}
			ok = t.topicSubscribe(topic, origin)
			if !ok {
				respondError(error.RuntimeFailure, "subscribe failed")
				return
			}

			response := message_event.Raw(
				fmt.Sprintf("~%s:%s ok", handle, command),
				"dev.sporeos.SPORE")
			event.Dispatch().Message.Notify(response)

		} else if command == "unsubscribe" || strings.HasSuffix(command, ".unsubscribe") {
			
			topic, ok := pm.Args["topic"]
			if !ok {
				witnessError(error.Missing, "mssing arg", out.Pair("arg", "topic"))
				respondError(error.Missing, "missing 'topic'")
				return
			}
			ok = t.topicUnsubscribe(topic, origin)
			if !ok {
				respondError(error.RuntimeFailure, "unsubscribe failed")
				return
			}

			response := message_event.Raw(
				fmt.Sprintf("~%s:%s ok", handle, command),
				"dev.sporeos.SPORE")
			event.Dispatch().Message.Notify(response)
		}
	}()
}

func (t *topics) Api() []string {
	return []string{
		"dev.sporeos.SPORE.topic.subscribe",
		"dev.sporeos.SPORE.topic.unsubscribe",
	}
}

func (t* topics) topicSubscribe(topic string, node string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	topics, ok := t.abbreviations[topic]
	if !ok {
		witness.Error(error.Missing, error.Mycelium, "topic not found")
		return false
	}
	if len(topics) > 1 {
		witness.Error(error.Collision, error.Mycelium, "topic collision")
		return false
	}

	fqTopic := topics[0]
	if slices.Contains(t.subscribers[fqTopic], node) {
		witness.Error(error.Collision, error.Mycelium, "already subscribed")
		return false
	}
	t.subscribers[fqTopic] = append(t.subscribers[fqTopic], node)
	return true
}

func (t* topics) topicUnsubscribe(topic string, node string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	topics, ok := t.abbreviations[topic]
	if !ok {
		witness.Error(error.Missing, error.Mycelium, "topic not found")
		return false
	}
	if len(topics) > 1 {
		witness.Error(error.Collision, error.Mycelium, "topic collision")
		return false
	}

	fqTopic := topics[0]
	if !slices.Contains(t.subscribers[fqTopic], node) {
		witness.Error(error.Missing, error.Mycelium, "not subscribed")
		return false
	}
	t.subscribers[fqTopic] = slices.DeleteFunc(t.subscribers[fqTopic], func(s string) bool { return s == node })
	return true
}

func (t *topics) OnNodeDisconnection(node string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for fqTopic, subs := range t.subscribers {
		t.subscribers[fqTopic] = slices.DeleteFunc(subs, func(s string) bool { return s == node })
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
