package bus

type Bus struct {
	api *api
	broadcast *broadcast
	witness *witness
}

func New() *Bus {
	return &Bus {
		api: newApi(),
		broadcast: newBroadcast(),
		witness: newWitness(),
	}
}

func (b *Bus) Close() {
	b.api.close()
	b.broadcast.close()
	b.witness.close()
}

func (b *Bus) Register(n node) {
	b.api.register(n)
	b.broadcast.register(n)
	b.witness.register(n)
}

func (b *Bus) Unregister(n node) {
	b.api.unregister(n)
	b.broadcast.unregister(n)
	b.witness.unregister(n)
}

func (b *Bus) Route(msg message) {
}

func (b *Bus) Witness(t WitnessType, msg string, n node) {
	b.witness.dispatch(t, msg, n)
}
