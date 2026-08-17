package bus

import (
	"spored/internal/iface"
	"spored/internal/utilities/error"
	"sync"
)

type broadcast struct {
	mu sync.RWMutex
	nodes map[string]INode
	subscriptions map[string][]string
}

func newBroadcast() *broadcast {
	return &broadcast {
		nodes: make(map[string]INode),
		subscriptions: make(map[string][]string),
	}
}

func (b *broadcast) close() {}

func (b *broadcast) register(n INode) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes[n.Id()] = n
}

func (b *broadcast) unregister(n INode) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.nodes, n.Id())
}

func (b *broadcast) subscribe(cast string, topic string) *error.Error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscriptions[cast] = append(b.subscriptions[cast], topic)
	return nil
}

func (b *broadcast) unsubscribe(cast string, topic string) *error.Error {
	b.mu.Lock()
	defer b.mu.Unlock()
	topics := b.subscriptions[cast]
	for i, t := range topics {
		if t == topic {
			b.subscriptions[cast] = append(topics[:i], topics[i+1:]...)
			break
		}
	}
	return nil
}

func (b *broadcast) broadcast(msg iface.Message) *error.Error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	topic := msg.Capability()
	subscribers, ok := b.subscriptions[topic]
	if !ok {
		return nil
	}
	for _, nodeid := range subscribers {
		node, ok := b.nodes[nodeid]
		if !ok {
			continue
		}
		err := node.Receive(msg)
		if err != nil {
			// TODO
			// how should I handle errors
		}
	}

	return nil
}
