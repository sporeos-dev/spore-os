// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package permissions

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

type ihyphae interface {}

type inodes interface {}

type ispore interface {}
