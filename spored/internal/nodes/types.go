package nodes

import "spored/internal/bus"

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	WitnessIn(msg string, id string)
	WitnessOut(msg string, id string)
	WitnessNode(msg string, id string)
}

type ihyphae interface {}

type ispore interface {}

