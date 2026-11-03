// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	sporeerror "spored/internal/utilities/error"
	"testing"
)

type permissionRequestTestPermissions struct {
	node       string
	capability string
	calls      int
}

func (p *permissionRequestTestPermissions) List(string) ([]string, *sporeerror.Error) {
	return nil, nil
}

func (p *permissionRequestTestPermissions) RegisterNode(*manifest.Manifest) {}

func (p *permissionRequestTestPermissions) Grant(string, string) *sporeerror.Error {
	return nil
}

func (p *permissionRequestTestPermissions) Revoke(string, string) *sporeerror.Error {
	return nil
}

func (p *permissionRequestTestPermissions) Request(node string, capability string, _ []string) (bool, *sporeerror.Error) {
	p.node = node
	p.capability = capability
	p.calls++
	return true, nil
}

func (p *permissionRequestTestPermissions) RequestPermit(node string, capability string, _ []string) (string, bool, *sporeerror.Error) {
	p.node = node
	p.capability = capability
	p.calls++
	return "permit", true, nil
}

func (p *permissionRequestTestPermissions) Can(iface.Message, string) (bool, *sporeerror.Error) {
	return true, nil
}

func TestPermissionRequestUsesRequestOrigin(t *testing.T) {
	permissions := &permissionRequestTestPermissions{}
	bus := &nodeCommandTestBus{responses: make(chan iface.Message, 1)}
	s := &Spore{
		manifest:    &manifest.Manifest{ID: "dev.sporeos.SPORE"},
		permissions: permissions,
		bus:         bus,
	}
	request := mustPermissionRequest(t,
		"dev.sporeos.SPORE.permission.request node=dev.sporeos.other capability=filesystem.read ~request",
		"SPORE.pipe:dev.sporeos.requester")

	if err := s.permissionRequest(request); err != nil {
		t.Fatalf("permissionRequest() returned an error: %v", err)
	}
	response := awaitNodeCommandResponse(t, bus)

	if permissions.calls != 1 {
		t.Fatalf("Request() calls = %d, want 1", permissions.calls)
	}
	if permissions.node != "dev.sporeos.requester" {
		t.Errorf("Request() node = %q, want request origin", permissions.node)
	}
	if permissions.capability != "filesystem.read" {
		t.Errorf("Request() capability = %q, want %q", permissions.capability, "filesystem.read")
	}
	if token, ok := response.Arg("permission.token"); !ok || token != "permit" {
		t.Errorf("permission response token = %q, present=%t", token, ok)
	}
}

func mustPermissionRequest(t *testing.T, raw string, cast string) iface.Message {
	t.Helper()

	request, ok := message.Request(raw, cast)
	if !ok {
		t.Fatalf("failed to parse request: %q", raw)
	}
	return request
}
