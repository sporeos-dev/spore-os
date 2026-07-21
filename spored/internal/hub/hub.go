package hub

import (
	"errors"
	"net"
	"os"
	"spored/internal/bus"
	"spored/internal/hyphae"
	"spored/internal/nodes"
	"spored/internal/pal"
	"spored/internal/spore"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

type Hub struct {
	bus *bus.Bus
	hyphae *hyphae.Hyphae
	nodes *nodes.Nodes
	spore *spore.Spore

	listener net.Listener
}

func New() (*Hub, *error.Error) {
	
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
		error.New(
			error.InitializationFailure,
			error.Hub,
			"Failed to listen on socket",
			out.Pair("error", err.Error()))
		return
	}
	h.listener = listener
	err = os.Chmod(pal.FileSocket(), 0777)
	if err != nil {
		error.New(
			error.InitializationFailure,
			error.Hub,
			"Failed to set permissions on socket",
			out.Pair("error", err.Error()))
		return;
	}
	defer h.listener.Close()
	
	h.nodes.Autostart()

	for {
		conn, err := h.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			error.New(
				error.ConnectionFailure,
				error.Hub,
				"Failed to accept new connection",
				out.Pair("error", err.Error()))
			continue
		}
		
		go h.nodes.HandleConnection(conn)
	}
}