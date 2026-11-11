// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"net"
	"spored/internal/base"
	"spored/internal/event"
	"spored/internal/manifest"
	"spored/internal/node"
	"spored/internal/registry"
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

type Pool struct {
	*base.Base
	index   atomic.Int64
	pending *await.Pending

	registry *registry.Registry

	mu    sync.RWMutex
	nodes map[string]*node.Single
}

type installation struct {
	manifest *manifest.Manifest
	registry *registry.Element
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
