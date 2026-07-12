package bus

type broadcast struct {
	
}

func newBroadcast() *broadcast {
	return &broadcast {}
}

func (b *broadcast) close() {}

func (b *broadcast) register(n inode) {}

func (b *broadcast) unregister(n inode) {}