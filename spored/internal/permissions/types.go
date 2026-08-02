package permissions

import (
	"spored/internal/bus"
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type Value string
const (
	Always Value = "always"
	Once Value = "once"
	No Value = "no"
	Never Value = "never"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	WitnessSpore(msg string)
	Request(msg message.Message) *error.Error
}

type ihyphae interface {}

type inodes interface {}

type ispore interface {}
