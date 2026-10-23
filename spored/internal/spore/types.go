// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package spore

import (
	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/nodes"
	"spored/internal/utilities/error"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	Subscribe(cast string, topic string) *error.Error
	Unsubscribe(cast string, topic string) *error.Error
	Response(msg iface.Message) *error.Error
	FullyQualifiedRequest(command string) string
	Witness(msg iface.Message)
}

type ihyphae interface{}

type inodes interface {
	GetNodes() []string
	GetManifest(nodeid string) *manifest.Manifest
	Install(path string, load bool, origin string) *error.Error
	Uninstall(nodeid string) *error.Error
	Reinstall(nodeid string, load bool, origin string) *error.Error
	Spawn(nodeid string) *error.Error
	Kill(nodeid string) *error.Error
	Reload(nodeid string) *error.Error
	GetState(nodeid string) (*nodes.State, *error.Error)
}

type ipermissions interface {
	List(nodeid string) ([]string, *error.Error)
	RegisterNode(manifest *manifest.Manifest)
	Grant(nodeid string, capability string) *error.Error
	Revoke(nodeid string, capability string) *error.Error
	Request(nodeid string, capability string, reasons []string) (bool, *error.Error)
	RequestPermit(nodeid string, capability string, reasons []string) (string, bool, *error.Error)
	Can(msg iface.Message, capability string) (bool, *error.Error)
}
