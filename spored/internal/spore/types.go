package spore

import (
	"spored/internal/bus"
	"spored/internal/manifest"
	"spored/internal/utilities/error"
)

type ibus interface {
	Subscribe(cast string, topic string) *error.Error
	Unsubscribe(cast string, topic string) *error.Error
	Register(node bus.INode)
	Unregister(node bus.INode)
}

type ihyphae interface {}

type inodes interface {
	GetNodes() []string
	GetManifest(nodeid string) *manifest.Manifest
	Install(path string) *error.Error
	Uninstall(nodeid string) *error.Error
	Spawn(nodeid string) *error.Error
	Kill(nodeid string) *error.Error
}

