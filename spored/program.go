// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"log/slog"
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
	p.hub = &hub.Hub{}
	if err := p.hub.Open(); err != nil {
		slog.Error("Failure to setup", "error", err)
		return err
	}
	return nil
}

// Stop is called by the service manager on SIGINT/SIGTERM or when
// `spored stop` is run. hub.Close() is synchronous cleanup.
func (p *program) Stop(s service.Service) error {
	p.hub.Close()
	return nil
}
