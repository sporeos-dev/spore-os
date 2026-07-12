package hub

import (
	"errors"
	"log/slog"
	"net"
	"os"
	"spored/internal/bus"
	"spored/internal/hyphae"
	"spored/internal/nodes"
	"spored/internal/pal"
	"spored/internal/spore"
)

type Hub struct {
	bus *bus.Bus
	hyphae *hyphae.Hyphae
	nodes *nodes.Nodes
	spore *spore.Spore

	listener net.Listener
}

func New() (*Hub, error) {
	
	h := &Hub{
		bus: bus.New(),
		hyphae: hyphae.New(),
		nodes: nodes.New(),
		spore: spore.New(),
	}

	h.bus.Set(h.hyphae, h.nodes, h.spore)
	h.hyphae.Set(h.bus, h.nodes, h.spore)
	h.nodes.Set(h.bus, h.hyphae, h.spore)
	h.spore.Set(h.bus, h.hyphae, h.nodes)

	go h.listen()

	return h, nil
}

func (h *Hub) Close() {
	h.bus.Close()
	h.hyphae.Close()
	h.nodes.Close()
	h.spore.Close()

	if h.listener != nil {
		h.listener.Close()
	}
	os.Remove(pal.FileSocket())
}

func (h *Hub) listen() {

	os.Remove(pal.FileSocket())
	listener, err := net.Listen("unix", pal.FileSocket())
	if err != nil {
		slog.Error("Failed to listen on socket", "error", err)
		return
	}
	h.listener = listener
	defer h.listener.Close()
	
	h.nodes.Autostart()

	for {
		conn, err := h.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			slog.Error("Failed to accept connection", "error", err)
			continue
		}
		
		go h.nodes.HandleConnection(conn)
	}
}