package bus

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type ihyphae interface {}

type inodes interface {
	Publish(message message.Message) *error.Error
}

type inode interface {
	Id() string
	IsWitness() bool
	Receive(message message.Message) *error.Error
}

type ispore interface {}
