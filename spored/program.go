// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"log/slog"
	"os"
	"spored/internal/hub"

	"github.com/kardianos/service"
)

// program implements service.Interface, bridging kardianos/service
// into the hub's Open/Close lifecycle.
type program struct {
	hub *hub.Hub
}

// Start is called by the service manager when the daemon starts.
// It must return quickly — hub.Open() already does this by launching
// its accept loop in a goroutine.
func (p *program) Start(s service.Service) error {
	slog.Debug("Starting program as a service")

	hub, ok := hub.New()
	if !ok {
		slog.Error("Failed to start hub")
		return nil
	}
	p.hub = hub
	return nil
}

// Stop is called by the service manager on SIGINT/SIGTERM or when
// `spored stop` is run. hub.Close() is synchronous cleanup.
func (p *program) Stop(s service.Service) error {
	slog.Debug("Stopping program as a service")

	p.hub.Close()
	return nil
}

// run/stop are convenience methods for running the hub in interactive mode
func (p *program) run() {
	slog.Debug("Starting program as a console app")

	hub, ok := hub.New()
	if !ok {
		slog.Error("Failed to start hub")
		os.Exit(1)
	}
	p.hub = hub

	select {}
}

func (p *program) stop() {
	slog.Debug("Stopping program as a console app")

	p.hub.Close()
}

