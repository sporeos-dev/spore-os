// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package hub

import (
	"errors"
	"net"
	"spored/internal/bus"
	"spored/internal/event"
	"spored/internal/hyphae"
	"spored/internal/mycelium"
	"spored/internal/nodes"
	"spored/internal/pal"
	"spored/internal/permissions"
	"spored/internal/spore"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/witness"
)

type Hub struct {
	bus         *bus.Bus
	hyphae      *hyphae.Hyphae
	nodes       *nodes.Nodes
	mycelium    *mycelium.Mycelium
	spore       *spore.Spore
	permissions *permissions.Manager

	listener net.Listener
}

func New() (*Hub, *error.Error) {

	h := &Hub{
		bus:         bus.New(),
		hyphae:      hyphae.New(),
		mycelium:    mycelium.New(),
		nodes:       nodes.New(),
		spore:       spore.New(),
		permissions: permissions.New(),
	}

	event.Init()
	h.mycelium.Open()

	h.bus.Set(h.hyphae, h.nodes, h.permissions, h.spore)
	h.hyphae.Set(h.bus, h.nodes, h.permissions, h.spore)
	h.nodes.Set(h.bus, h.hyphae, h.permissions, h.spore)
	h.permissions.Set(h.bus, h.hyphae, h.nodes, h.spore)
	h.spore.Set(h.bus, h.hyphae, h.nodes, h.permissions)

	go h.listen()
	return h, nil
}

func (h *Hub) Close() {

	h.mycelium.Close()
	h.bus.Close()
	h.hyphae.Close()
	h.nodes.Close()
	h.permissions.Close()
	h.spore.Close()

	if h.listener != nil {
		h.listener.Close()
	}
	if err := pal.RemoveLocalEndpoint(); err != nil {
		witness.Error(
			error.ConnectionFailure,
			error.Hub,
			"Failed to remove local endpoint",
			out.Pair("error", err.Error()))
	}
}

func (h *Hub) listen() {

	if err := pal.RemoveLocalEndpoint(); err != nil {
		witness.Error(
			error.InitializationFailure,
			error.Hub,
			"Failed to remove local endpoint",
			out.Pair("error", err.Error()))
		return
	}
	listener, err := pal.ListenLocal()
	if err != nil {
		witness.Error(
			error.InitializationFailure,
			error.Hub,
			"Failed to listen on socket",
			out.Pair("error", err.Error()))
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
			witness.Error(
				error.ConnectionFailure,
				error.Hub,
				"Failed to accept new connection",
				out.Pair("error", err.Error()))
			continue
		}

		go h.nodes.HandleConnection(conn)
	}
}
