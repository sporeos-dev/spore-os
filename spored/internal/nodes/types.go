package nodes

import (
	"spored/internal/bus"
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	WitnessIn(msg string, id string)
	WitnessOut(msg string, id string)
	WitnessNode(msg string, id string)
	Request(msg message.Message) *error.Error
	Response(msg message.Message) *error.Error
	Broadcast(msg message.Message) *error.Error
}

type ihyphae interface {}

type ispore interface {}

