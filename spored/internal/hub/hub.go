package hub

import (
	"spored/internal/bus"
	"spored/internal/hyphae"
	"spored/internal/nodes"
	"spored/internal/spore"
)

type Hub struct {
	bus *bus.Bus
	hyphae *hyphae.Hyphae
	nodes *nodes.Nodes
	spore *spore.Spore
}

func New() (*Hub, error) {
	hub := &Hub{
		bus: bus.New(),
		hyphae: hyphae.New(),
		nodes: nodes.New(),
		spore: spore.New(),
	}

	return hub, nil
}

func (h *Hub) Close() {
	h.bus.Close()
	h.hyphae.Close()
	h.nodes.Close()
	h.spore.Close()
}