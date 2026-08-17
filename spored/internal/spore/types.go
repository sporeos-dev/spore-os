package spore

import (
	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/permissions"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	Subscribe(cast string, topic string) *error.Error
	Unsubscribe(cast string, topic string) *error.Error
	Response(msg iface.Message) *error.Error
}

type ihyphae interface {}

type inodes interface {
	GetNodes() []string
	GetManifest(nodeid string) *manifest.Manifest
	Install(path string) *error.Error
	Uninstall(nodeid string) *error.Error
	Spawn(nodeid string) *error.Error
	Kill(nodeid string) *error.Error
	GetState(nodeid string) ([]out.IOut, *error.Error)
}

type ipermissions interface {
	List(nodeid string) ([]string, *error.Error)
	Grant(nodeid string, capability string) *error.Error
	Revoke(nodeid string, capability string) *error.Error
	Request(nodeid string, capability string, reasons []string) (permissions.Value, *error.Error)
	Can(nodeid string, capability string) bool
}

