package bus

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type Bus struct {
	api *api
	broadcast *broadcast
	responses *responses
	witness *witness
	
	hyphae ihyphae
	nodes inodes
	spore ispore
}

func New() *Bus {
	return &Bus {
		api: newApi(),
		broadcast: newBroadcast(),
		responses: newResponses(),
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
	b.responses.close()
	b.witness.close()
}

func (b *Bus) Register(n INode) {
	b.api.register(n)
	b.broadcast.register(n)
	b.responses.register(n)
	b.witness.register(n)
}

func (b *Bus) Unregister(n INode) {
	b.api.unregister(n)
	b.broadcast.unregister(n)
	b.responses.unregister(n)
	b.witness.unregister(n)
}

func (b *Bus) Broadcast(msg message.Message) *error.Error {
	return b.broadcast.broadcast(msg)
}

func (b *Bus) Request(msg message.Message) *error.Error {
	err := b.responses.request(msg)
	if err != nil {
		return err
	}
	return b.api.request(msg)
}

func (b *Bus) Response(msg message.Message) *error.Error {
	return b.responses.response(msg)
}

func (b *Bus) WitnessIn(message string, id string) {
	b.witness.in(message, id)
}

func (b *Bus) WitnessOut(message string, id string) {
	b.witness.out(message, id)
}

func (b *Bus) WitnessSpore(message string) {
	b.witness.spore(message)
}

func (b *Bus) WitnessNode(message string, id string) {
	b.witness.node(message, id)
}

func (b *Bus) Subscribe(cast string, topic string) *error.Error {
	return b.broadcast.subscribe(cast, topic)
}

func (b *Bus) Unsubscribe(cast string, topic string) *error.Error {
	return b.broadcast.unsubscribe(cast, topic)
}
