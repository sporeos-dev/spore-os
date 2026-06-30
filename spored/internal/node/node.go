// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package node

import (
	"errors"
	"fmt"
	"net"
	"spored/internal/connection"
	"spored/internal/interfaces"
	"spored/internal/manifest"
	"spored/internal/message"
	"sync"
	"time"
)

const handshakeTimeout = 5 * time.Second

type Router interface {
	Route(msg message.Message) error
	AddRoute(command string, nodeid string)
	PurgeNode(nodeID string)
}

type Node struct {
	Router             Router
	WitnessDispatcher  interfaces.Witness
	Broadcaster        interfaces.Broadcaster
	connection *connection.Connection
	Manifest *manifest.Manifest

	mu sync.Mutex
	jsonHandles map[string]bool
}

func (n *Node) Open(path string) (string, error) {
	manifest, err := manifest.LoadManifest(path)
	if err != nil {
		return "", err
	}
	n.Manifest = manifest
	for _, command := range n.Manifest.Api {
		n.Router.AddRoute(command.Name, n.Manifest.ID)
	}
	n.connection = &connection.Connection{}
	n.jsonHandles = make(map[string]bool)
	return n.Manifest.ID, nil
}

func (n *Node) Connect(conn net.Conn) error {
	err := n.connection.Open(conn)
	if err != nil {
		return err
	}
	go n.connection.Run(n)
	return nil
}

func (n *Node) Disconnect() error {
	n.connection.Close()
	return nil
}

// Disconnected is called by connection.Run when the node's socket connection
// closes (either cleanly via EOF or after an unrecoverable read error). It
// notifies the router so any in-flight handles waiting on this node can be
// cleaned up immediately rather than leaking until the caster retries.
func (n *Node) Disconnected() {
	n.Router.PurgeNode(n.Manifest.ID)
	if n.Broadcaster != nil {
		n.Broadcaster.PurgeNode(n.Manifest.ID)
	}
}

func (n *Node) Id() string {
	return n.Manifest.ID
}

func (n *Node) Route(msg message.Message) error {
	if msg.IsPublish() {
		if pub, ok := msg.(*message.Publish); ok && n.Broadcaster != nil {
			return n.Broadcaster.Publish(pub)
		}
	}
	if msg.IsCast() {
		cast := msg.(*message.Cast)
		if cast.HasFlag("json") {
			n.mu.Lock()
			n.jsonHandles[cast.Handle()] = true
			n.mu.Unlock()
		}
	}
	err := n.Router.Route(msg)
	if err != nil {
		return err
	}
	return nil
}

func (n *Node) Send(msg message.Message) error {
	if n.connection == nil {
		return errors.New("node not connected")
	}
	if msg.IsCapture() {
		handle := msg.Handle()
		n.mu.Lock()
		wantsJson := n.jsonHandles[handle]
		delete(n.jsonHandles, handle)
		n.mu.Unlock()
		if wantsJson {
			return n.connection.SendRaw(msg.ToJsonString())
		}
	}
	return n.connection.Send(msg)
}

func (n *Node) SendWitness(msg string) {
	if n.connection == nil {
		return
	}
	n.connection.SendRaw(msg)
}

func (n *Node) WitnessIncoming(msg string, node string, index int64) {
	msg = fmt.Sprintf("%s cast=%s i=%d", msg, node, index)
	n.WitnessDispatcher.Incoming(msg)
}

func (n *Node) WitnessOutgoing(msg string, node string, index int64) {
	msg = fmt.Sprintf("%s capture=%s i=%d", msg, node, index)
	n.WitnessDispatcher.Outgoing(msg)
}

func (n *Node) WitnessSpore(msg string) {
	n.WitnessDispatcher.Spore(msg)
}

func (n *Node) WitnessNode(msg string) {
	n.WitnessDispatcher.Node(msg, n.Manifest.ID)
}

func (n *Node) GetManifest() *manifest.Manifest {
	return n.Manifest
}

func (n *Node) IsConnected() bool {
	return n.connection.IsConnected()
}

func (n *Node) GetPID() int {
	return n.connection.GetPID()
}

