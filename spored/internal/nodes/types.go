// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/registry"
	"spored/internal/utilities/error"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	Request(msg iface.Message) *error.Error
	Response(msg iface.Message) *error.Error
	Broadcast(msg iface.Message) *error.Error
	Pipe(msg iface.Message, node bus.INode)
	Witness(msg iface.Message)
}

type ihyphae interface {
	ManifestRead(path string) (string, *error.Error)
	HashFile(path string) (string, *error.Error)
	HashBinary(pid int) (string, *error.Error)
	PrepareForInstallation(path string) (manifest *manifest.Manifest, registryElement *registry.Element, err *error.Error)
	Spawn(path string) *error.Error
	Kill(pid int) *error.Error
}

type ipermissions interface {
	Request(nodeid string, capability string, reasons []string) (bool, *error.Error)
	Can(nodeid string, capability string) bool
	RegisterNode(man *manifest.Manifest)
}

type ispore interface{}
