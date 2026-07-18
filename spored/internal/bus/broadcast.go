package bus

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"sync"
)

type broadcast struct {
	mu sync.RWMutex
	nodes map[string]inode
	subscriptions map[string][]string // map[topic][]nodeid
}

func newBroadcast() *broadcast {
	return &broadcast {
		nodes: make(map[string]inode),
		subscriptions: make(map[string][]string),
	}
}

func (b *broadcast) close() {}

func (b *broadcast) register(n inode) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nodes[n.Id()] = n
}

func (b *broadcast) unregister(n inode) {
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

func (b *broadcast) publish(bus Bus, message message.Message) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	topic := message.Topic()
	subscribers, ok := b.subscriptions[topic]
	if !ok {
		return
	}
	for _, nodeID := range subscribers {
		node, ok := b.nodes[nodeID]
		if !ok {
			continue
		}
		err := node.Receive(message)
		if err != nil {
			bus.Witness(message.Spore, )
		}
	}
}
