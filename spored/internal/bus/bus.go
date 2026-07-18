package bus

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type Bus struct {
	api *api
	broadcast *broadcast
	witness *witness
	
	hyphae ihyphae
	nodes inodes
	spore ispore
}

func New() *Bus {
	return &Bus {
		api: newApi(),
		broadcast: newBroadcast(),
		witness: newWitness(),
	}
}

func (b *Bus) Set(hyphae ihyphae, nodes inodes, spore ispore) {
	b.hyphae = hyphae
	b.nodes = nodes
	b.spore = spore
}

func (b *Bus) Close() {
	b.api.close()
	b.broadcast.close()
	b.witness.close()
}

func (b *Bus) Register(n inode) {
	b.api.register(n)
	b.broadcast.register(n)
	b.witness.register(n)
}

func (b *Bus) Unregister(n inode) {
	b.api.unregister(n)
	b.broadcast.unregister(n)
	b.witness.unregister(n)
}

func (b *Bus) Route(msg *message.Message) {
}

func (b *Bus) Witness(message message.Message) {
	b.witness.dispatch(t, message, n)
}

func (b *Bus) Subscribe(cast string, topic string) *error.Error {
	return b.broadcast.subscribe(cast, topic)
}

func (b *Bus) Unsubscribe(cast string, topic string) *error.Error {
	return b.broadcast.unsubscribe(cast, topic)
}
