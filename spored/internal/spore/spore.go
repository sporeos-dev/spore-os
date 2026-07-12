package spore

type Spore struct {
	bus ibus
	hyphae ihyphae
	nodes inodes
}

func New() *Spore {
	return &Spore{}
}

func (s *Spore) Set(bus ibus, hyphae ihyphae, nodes inodes) {
	s.bus = bus
	s.hyphae = hyphae
	s.nodes = nodes
}

func (s *Spore) Close() {}
