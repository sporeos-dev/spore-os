package bus

import (
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type INode interface {
	Id() string
	IsConnected() bool
	IsWitness() bool
	GetManifest() *manifest.Manifest
	Receive(message message.Message) *error.Error
	Witness(message message.Message)
}

type ihyphae interface {}

type inodes interface {}

type ipermissions interface {}

type ispore interface {}
