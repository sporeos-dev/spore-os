// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"net"
	"spored/internal/base"
	"spored/internal/event"
	registry_event "spored/internal/event/registry"
	"spored/internal/manifest"
	"spored/internal/node"
	reg_el "spored/internal/registry/element"
	"spored/internal/utilities/await"
	"spored/internal/utilities/out"
	"spored/internal/witness"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"spored/internal/utilities/error"
)

const handshakeTimeout = 5 * time.Second
var _ registry_event.IRegistryAnnouncementHandler = (*Pool)(nil)
var _ registry_event.IRegistryWithdrawalHandler = (*Pool)(nil)

type Pool struct {
	*base.Base
	index   atomic.Int64
	pending *await.Pending

	mu    sync.RWMutex
	nodes map[string]*node.Single
}

type installation struct {
	manifest *manifest.Manifest
	registry *reg_el.Element
}

func New() *Pool {
	return &Pool{
		Base:     base.New().SetName("nodepool"),
		pending:  await.New(error.NodePool),
		nodes:    make(map[string]*node.Single),
	}
}

func (p *Pool) Open() {
	event.Subscribe(p)
}

func (p *Pool) Close() {
	event.Unsubscribe(p)
}

func (p *Pool) HandleConnection(conn net.Conn) {

	erro := conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetReadDeadline(time.Time{}) //nolint:errcheck
	if erro != nil {
		witness.Error(error.HandshakeDenial, error.Node, "read deadline", out.Pair("error", erro.Error()))
	}

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	nodeid, erro := reader.ReadString('\n')
	if erro != nil {
		witness.Error(error.HandshakeDenial, error.Node, "handshake read failed", out.Pair("error", erro.Error()))
		writer.WriteString("FAIL handshake read failed\n")
		writer.Flush()
		conn.Close()
		return
	}

	nodeid = strings.TrimSpace(nodeid)
	witness.Spore("node connecting", out.Pair("node", nodeid))

	p.mu.RLock()
	defer p.mu.RUnlock()
	node, ok := p.nodes[nodeid]
	if !ok {
		witness.Error(error.HandshakeDenial, error.Node, "node not installed", out.Pair("node", nodeid))
		writer.WriteString("FAIL node not installed\n")
		writer.Flush()
		conn.Close()
		return
	}

	// Handle the connection with the node
	// failures will be handled in the function
	// (including witness & replies)
	node.HandleConnection(conn, reader, writer)
}

// registry announcment
func (p *Pool) OnRegistryAnnouncement(ctx registry_event.IRegistryContext) {
	el := ctx.Registry()
	witness.Spore("registry announcement", out.Pair("node", el.ID))

	{
		p.mu.Lock()

		_, ok := p.nodes[el.ID]
		if ok {
			p.mu.Unlock()
			witness.Error(error.Collision, error.NodePool, "cannot add to registry", out.Pair("node", el.ID))
			return
		}

		node := node.New(el)
		p.nodes[el.ID] = node
		
		p.mu.Unlock()
	}

	witness.Spore("node added to registry", out.Pair("node", el.ID))
}

// registry withdawal
func (p *Pool) OnRegistryWithdrawal(ctx registry_event.IRegistryContext) {
	el := ctx.Registry()
	witness.Spore("registry withdrawal", out.Pair("node", el.ID))

	{
		p.mu.Lock()

		node, ok := p.nodes[el.ID]
		if ok {
			node.Close()
			delete(p.nodes, el.ID)
			p.mu.Unlock()
			witness.Spore("node removed from registry", out.Pair("node", el.ID))
			return
		}

		p.mu.Unlock()
	}

	witness.Error(error.Missing, error.NodePool, "cannot withdraw from registry", out.Pair("node", el.ID))
}