package nodes

import (
	"spored/internal/bus"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/permissions"
	"spored/internal/registry"
	"spored/internal/utilities/error"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	Request(msg message.Message) *error.Error
	Response(msg message.Message) *error.Error
	Broadcast(msg message.Message) *error.Error
	Witness(msg message.Message)
}

type ihyphae interface {
	ManifestRead(path string) (string, *error.Error)
	HashFile(path string) (string, *error.Error)
	PrepareForInstallation(path string) (manifest *manifest.Manifest, registryElement *registry.Element, err *error.Error)
	Spawn(path string) *error.Error
	Kill(pid int) *error.Error
}

type ipermissions interface {
	Request(nodeid string, capability string, reasons []string) (permissions.Value, *error.Error)
	Can(nodeid string, capability string) bool
}

type ispore interface {}

