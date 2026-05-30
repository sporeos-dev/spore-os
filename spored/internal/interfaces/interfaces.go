// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package interfaces

import (
	"spored/internal/manifest"
	"spored/internal/message"
)

// Hub is the central interface that router and spore depend on.
// Hub is the only thing that imports everything else, preventing circular imports.
type Hub interface {
	ListNodes() []string
	GetNode(nodeid string) (Node, error)
	AddNode(path string) error
	RemoveNode(nodeid string) error
	SpawnNode(nodeid string) error
	KillNode(nodeid string) error
}

// Node is what router and spore use to send messages to applications.
type Node interface {
	Send(msg message.Message) error
	SendWitness(msg string)
	GetManifest() *manifest.Manifest
	IsConnected() bool
	GetPID() int
}

// Registry is what spore uses to manage installed nodes.
type Registry interface {
	Open() error
	Paths() []string
	Add(path string) error
	Remove(path string) error
}

// Router is what hub uses to route messages.
type Router interface {
	Open(hub Hub, spore Spore)
	AddRoute(command string, nodeid string)
	Route(msg message.Message) error
	GetRoute(command string) (string, error)
	ListCommands(node string) []string
	PurgeNode(nodeID string)
}

// Spore is what hub uses to handle SPORE.* commands.
type Spore interface {
	Open(hub Hub, registry Registry, router Router, witness Witness)
	Command(incoming message.Message) (message.Message, error)
}

type Witness interface {
	Register(node Node)
	Unregister(id string)
	Incoming(msg string)
	Outgoing(msg string)
	Spore(msg string)
	Node(msg string, cast string)
}