package hyphae

type Hyphae struct {
	bus ibus
	nodes inodes
	spore ispore
}

func New() *Hyphae {
	return &Hyphae {}
}

func (h *Hyphae) Set(bus ibus, nodes inodes, spore ispore) {
	h.bus = bus
	h.nodes = nodes
	h.spore = spore
}

func (h *Hyphae) Close() {}
