package bus

import (
	"spored/internal/iface"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"strings"
	"sync"
)

type broadcast struct {
	mu sync.RWMutex
	nodes map[string]INode
	topics *abbreviationIndex
	subscriptions map[string][]string // topic fqname -> subscriber node ids
}

func newBroadcast() *broadcast {
	return &broadcast {
		nodes: make(map[string]INode),
		topics: newAbbreviationIndex(),
		subscriptions: make(map[string][]string),
	}
}

func (b *broadcast) close() {}

func (b *broadcast) register(n INode) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nodes[n.Id()] = n
	manifest := n.GetManifest()
	if manifest != nil {
		for _, topic := range manifest.Topics {
			b.topics.add(n.Id() + "." + topic.Name)
		}
	}
}

func (b *broadcast) unregister(n INode) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.nodes, n.Id())
	manifest := n.GetManifest()
	if manifest != nil {
		for _, topic := range manifest.Topics {
			b.topics.remove(n.Id() + "." + topic.Name)
		}
	}
}

// resolveTopic maps a (possibly abbreviated) topic name to its canonical
// fqname, or an error if it's unknown or ambiguous.
func (b *broadcast) resolveTopic(topic string) (string, *error.Error) {
	fqname, candidates, ok := b.topics.resolve(topic)
	if ok {
		return fqname, nil
	}
	if len(candidates) > 0 {
		return "", error.New(
			error.Collision,
			error.Bus,
			"ambiguous topic, use a more qualified name",
			out.Pair("topic", topic),
			out.Pair("candidates", strings.Join(candidates, ", ")))
	}
	return "", error.New(
		error.Missing,
		error.Bus,
		"topic not found",
		out.Pair("topic", topic))
}

func (b *broadcast) subscribe(cast string, topic string) *error.Error {
	b.mu.Lock()
	defer b.mu.Unlock()

	fqname, err := b.resolveTopic(topic)
	if err != nil {
		return err
	}

	b.subscriptions[fqname] = append(b.subscriptions[fqname], cast)
	return nil
}

func (b *broadcast) unsubscribe(cast string, topic string) *error.Error {
	b.mu.Lock()
	defer b.mu.Unlock()

	fqname, err := b.resolveTopic(topic)
	if err != nil {
		return err
	}

	subscribers := b.subscriptions[fqname]
	for i, s := range subscribers {
		if s == cast {
			b.subscriptions[fqname] = append(subscribers[:i], subscribers[i+1:]...)
			break
		}
	}
	return nil
}

func (b *broadcast) broadcast(msg iface.Message) *error.Error {
	b.mu.RLock()

	topic := msg.Capability()
	fqname, candidates, ok := b.topics.resolve(topic)
	if !ok {
		b.mu.RUnlock()
		if len(candidates) > 0 {
			return error.New(
				error.Collision,
				error.Bus,
				"ambiguous topic, use a more qualified name",
				out.Pair("topic", topic),
				out.Pair("candidates", strings.Join(candidates, ", "))).
				WithMessage(msg)
		}
		// Nobody could have subscribed to an unresolvable topic.
		return nil
	}

	subscribers, ok := b.subscriptions[fqname]
	if !ok {
		b.mu.RUnlock()
		return nil
	}
	targets := make([]INode, 0, len(subscribers))
	for _, nodeid := range subscribers {
		if node, ok := b.nodes[nodeid]; ok {
			targets = append(targets, node)
		}
	}

	b.mu.RUnlock()
	for _, node := range targets {
		err := node.Receive(msg)
		if err != nil {
			// TODO
			// how should I handle errors
		}
	}

	return nil
}
