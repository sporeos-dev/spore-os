package bus

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type INode interface {
	Id() string
	IsConnected() bool
	IsWitness() bool
	Receive(message message.Message) *error.Error
}

type ihyphae interface {}

type inodes interface {}


type ispore interface {}
