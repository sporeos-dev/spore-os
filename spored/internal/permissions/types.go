package permissions

import (
	"spored/internal/bus"
	"spored/internal/iface"
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
	Request(msg iface.Message) *error.Error
}

type ihyphae interface {}

type inodes interface {}

type ispore interface {}
