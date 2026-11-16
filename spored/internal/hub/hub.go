// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package hub

import (
	"errors"
	"log/slog"
	"net"
	"spored/internal/event"
	manifest_event "spored/internal/event/manifest"
	"spored/internal/manifest"
	"spored/internal/manifestloader"
	"spored/internal/mycelium"
	nodes "spored/internal/nodepool"
	"spored/internal/pal"
	"spored/internal/persist"
	"spored/internal/registry"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"spored/internal/witness"
)

type Hub struct {
	mycelium *mycelium.Mycelium
	nodepool *nodes.Pool
	persistence *persist.Storage
	registry *registry.Registry

	listener net.Listener
	sporeManifest *manifest.Manifest
}

func New() (*Hub, bool) {

	h := &Hub{
		mycelium: mycelium.New(),
		nodepool: nodes.New(),
		persistence: persist.New(),
		registry: registry.New(),
		sporeManifest: nil,
	}

	event.Init()
	h.mycelium.Open()
	h.nodepool.Open()
	h.persistence.Open()
	h.registry.Open()

	if !h.loadAndAnnounceSporeManifest() {
		h.mycelium.Close()
		h.nodepool.Close()
		h.persistence.Close()
		h.registry.Close()
		return nil, false
	}

	go h.listen()
	h.registry.Load()

	return h, true
}
func (h *Hub) Close() {

	h.mycelium.Close()
	h.nodepool.Close()
	h.persistence.Close()
	h.registry.Close()

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

		go h.nodepool.HandleConnection(conn)
	}
}

func (h *Hub) loadAndAnnounceSporeManifest() bool {
	h.sporeManifest = &manifest.Manifest {
		Status: status.New(),
		Path: pal.FileSporeManifest(),
		ExpectedChecksum: "n/a",
		Notifier: h,
	}
	h.sporeManifest.Status.Set(status.Verified)
	if !manifestloader.Load(h.sporeManifest) {
		return false
	}
	slog.Debug("Spore manifest loaded")
	return true
}

// inotifier
func (h *Hub) Announce(m *manifest.Manifest) {
	ctx := manifest_event.New(m)
	event.Dispatch().ManifestAnnouncement.Notify(ctx)
}

func (h *Hub) Withdraw(m *manifest.Manifest) {
	ctx := manifest_event.New(m)
	event.Dispatch().ManifestWithdrawal.Notify(ctx)
}
