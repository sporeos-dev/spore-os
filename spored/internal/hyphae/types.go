// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package hyphae

import (
	"spored/internal/bus"
	"spored/internal/iface"
	"spored/internal/utilities/error"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	Request(msg iface.Message) *error.Error
}

type inodes interface {}

type ipermissions interface {}

type ispore interface {}
