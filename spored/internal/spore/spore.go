// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"log/slog"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/pal"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
)

type Spore struct {
	manifest *manifest.Manifest

	bus ibus
	hyphae ihyphae
	permissions ipermissions
	nodes inodes
}
	
func New() *Spore {
	m := &manifest.Manifest{
		Status: status.New(),
		Path: pal.FileSporeManifest(),
		ExpectedChecksum: "n/a",
	}
	m.Load()
	m.Status.Set(status.Verified)
	slog.Debug("Spore manifest loaded.", "manifest", m.Path)

	return &Spore{
		manifest: m,
	}
}

func (s *Spore) Set(bus ibus, hyphae ihyphae, nodes inodes, permissions ipermissions) {
	s.bus = bus
	s.hyphae = hyphae
	s.nodes = nodes
	s.permissions = permissions

	s.bus.Register(s)
}

func (s *Spore) Close() {
	s.bus.Unregister(s)
}

//
//
// INode
//

func (s *Spore) Id() string {
	return s.manifest.ID
}

func (s *Spore) IsConnected() bool {
	return true
}

func (s *Spore) IsWitness() bool {
	return false
}

func (s *Spore) GetManifest() *manifest.Manifest {
	return s.manifest
}

func (s *Spore) Receive(request iface.Message) *error.Error {
	capability := request.Capability()
	fqCap := s.bus.FullyQualifiedRequest(capability)

	switch fqCap {
	case "dev.sporeos.SPORE.help": return s.help(request)
	case "dev.sporeos.SPORE.state": return s.state(request)
	case "dev.sporeos.SPORE.list": return s.list(request)
	case "dev.sporeos.SPORE.complete": return s.complete(request)
	case "dev.sporeos.SPORE.hint": return s.hint(request)
		
	case "dev.sporeos.SPORE.node.install": return s.nodeInstall(request)
	case "dev.sporeos.SPORE.node.uninstall": return s.nodeUninstall(request)
	case "dev.sporeos.SPORE.node.spawn": return s.nodeSpawn(request)
	case "dev.sporeos.SPORE.node.kill": return s.nodeKill(request)

	case "dev.sporeos.SPORE.topic.subscribe": return s.topicSubscribe(request)
	case "dev.sporeos.SPORE.topic.unsubscribe": return s.topicUnsubscribe(request)

	case "dev.sporeos.SPORE.permission.request": return s.permissionRequest(request)
	case "dev.sporeos.SPORE.permission.grant": return s.permissionGrant(request)
	case "dev.sporeos.SPORE.permission.revoke": return s.permissionRevoke(request)

	case "dev.sporeos.SPORE.security.keyring.grant": return s.securityKeyringGrant(request)
	case "dev.sporeos.SPORE.security.keyring.revoke": return s.securityKeyringRevoke(request)
	case "dev.sporeos.SPORE.security.signature.sign": return s.securitySignatureSign(request)
	case "dev.sporeos.SPORE.security.signature.verify": return s.securitySignatureVerify(request)
	}

	return error.New(
		error.Missing,
		error.Spore,
		"spore command unhandled",
		out.Pair("command", request.Capability())).
		WithMessage(request)
}

func (s *Spore) Witness(message iface.Message) {}
