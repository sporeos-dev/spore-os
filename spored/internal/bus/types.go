// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package bus

import (
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/utilities/error"
)

type INode interface {
	Id() string
	IsConnected() bool
	IsWitness() bool
	GetManifest() *manifest.Manifest
	Receive(message iface.Message) *error.Error
	Witness(message iface.Message)
}

type ihyphae interface {}

type inodes interface {}

type ipermissions interface {}

type ispore interface {}
