package hyphae

import (
	"spored/internal/bus"
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	Witness(msg message.Message)
	Request(msg message.Message) *error.Error
}

type inodes interface {}

type ipermissions interface {}

type ispore interface {}
