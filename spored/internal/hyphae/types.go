package hyphae

import (
	"sync"

	"spored/internal/bus"
	"spored/internal/message"
	"spored/internal/utilities/error"
)

type ibus interface {
	Register(node bus.INode)
	Unregister(node bus.INode)
	WitnessSpore(msg string)
	Request(msg message.Message) *error.Error
}

type inodes interface {}

type ispore interface {}

type pending struct {
	mu      sync.Mutex
	channels map[string]chan message.Message
}
