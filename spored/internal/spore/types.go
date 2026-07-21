package spore

import (
	"spored/internal/manifest"
	"spored/internal/utilities/error"
)

type ibus interface {
	Subscribe(cast string, topic string) *error.Error
	Unsubscribe(cast string, topic string) *error.Error
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

type icast interface {
	Get() string

	Command() string
	Cast() string
	Arg(key string) (string, *error.Error)
	ArgIf(key string, ifnot string) string
	Flag(flag string) bool
}

type icapture interface {}


