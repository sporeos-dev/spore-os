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
