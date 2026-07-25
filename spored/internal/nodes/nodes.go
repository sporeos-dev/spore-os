package nodes

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"spored/internal/manifest"
	"spored/internal/registry"
	"spored/internal/utilities/file"
	"spored/internal/utilities/out"
	"strings"
	"sync"
	"time"

	"spored/internal/utilities/error"
)

const handshakeTimeout = 5 * time.Second

type Nodes struct {
	registry *registry.Registry

	mu sync.RWMutex
	nodes map[string]*node

	bus ibus
	hyphae ihyphae
	spore ispore
}

func New() *Nodes {
	nodes := &Nodes{
		registry: registry.New(),
		nodes:   make(map[string]*node),
	}

	nodes.mu.Lock()
	defer nodes.mu.Unlock()

	for _, el := range nodes.registry.Elements {
		manifest := manifest.New(el)
		n := newNode(el, manifest)
		nodes.nodes[n.registry.ID] = n
	}

	return nodes
}

func (n *Nodes) Set(bus ibus, hyphae ihyphae, spore ispore) {
	n.bus = bus
	n.hyphae = hyphae
	n.spore = spore

	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, el := range n.nodes {
		el.setBus(bus)
	}
}

func (n *Nodes) Close() {
	n.registry.Close()
}

func (n *Nodes) Autostart() {
	
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, el := range n.nodes {
		if el.manifest.Autostart {
			err := n.Spawn(el.registry.ID)
			if err != nil {
				error.New(
					error.Generic,
					error.Node,
					"autostart failure",
					out.Pair("node", el.registry.ID))
			}
		}
	}
}

func (n *Nodes) HandleConnection(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetReadDeadline(time.Time{})

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	nodeid, err := reader.ReadString('\n')
	if err != nil {
		sperr := error.New(
			error.HandshakeDenial,
			error.Node,
			"failed initial handshake read",
			out.Pair("error", err.Error()))
		writer.WriteString(sperr.Wire())
		writer.Flush()
		conn.Close()
		return
	}

	n.bus.WitnessSpore(fmt.Sprintf("Node connection attempted: %s", nodeid))

	nodeid = strings.TrimSpace(nodeid)
	n.mu.RLock()
	defer n.mu.RUnlock()
	node, ok := n.nodes[nodeid]
	if !ok {
		sperr := error.New(
			error.HandshakeDenial,
			error.Node,
			"node not installed",
			out.Pair("node", nodeid))
		writer.WriteString(sperr.Wire())
		writer.Flush()
		conn.Close()
		return
	}

	// Handle the connection with the node
	sp_err := node.handleConnection(conn, reader, writer)
	if sp_err != nil {
		writer.WriteString(sp_err.Wire())
		writer.Flush()
		conn.Close()
		return;
	}
}

//
//
// external
// interfaces
//

func (n *Nodes) GetNodes() []string {

	n.mu.RLock()
	defer n.mu.RUnlock()

	nodeids := make([]string, 0, len(n.nodes))
	for nodeid := range n.nodes {
		nodeids = append(nodeids, nodeid)
	}

	return nodeids
}

func (n *Nodes) GetManifest(nodeid string) *manifest.Manifest {
	nodeid = n.resolveId(nodeid)

	n.mu.RLock()
	defer n.mu.RUnlock()

	node, ok := n.nodes[nodeid]
	if !ok { return nil }
	
	return node.manifest
}

func (n *Nodes) Install(path string) *error.Error {

	if !file.Exists(path) {
		return error.New(
			error.Missing, 
			error.Node,
			"unable to install due to bad path",
			out.Pair("path", path))
	}

	// readable, handle via spore
	if file.IsReadable(path) {
		manifest := manifest.ManifestFromPath(path)
		if manifest == nil {
			return error.New(
				error.Generic,
				error.Node,
				"read failure",
				out.Pair("path", path))
		}

		registryElement := registry.ElementFromManifest(manifest)
		if registryElement == nil {
			return error.New(
				error.Generic,
				error.Node,
				"failed to create registry element",
				out.Pair("path", path))
		}

		err := n.registry.Add(registryElement)
		if err != nil {
			return err
		}

		node := newNode(registryElement, manifest)
		node.setBus(n.bus)
		n.nodes[node.registry.ID] = node

	// not readable, handle via hyphae
	} else {

		manifest, registry, err := n.hyphae.PrepareForInstallation(path)
		if err != nil {
			return err
		}

		err = n.registry.Add(registry)
		if err != nil {
			return err
		}

		node := newNode(registry, manifest)
		node.setBus(n.bus)
		n.nodes[node.registry.ID] = node
	}

	return nil
}

func (n *Nodes) Uninstall(nodeid string) *error.Error {
	nodeid = n.resolveId(nodeid)
	
	n.mu.Lock()
	defer n.mu.Unlock()

	node, ok := n.nodes[nodeid]
	if !ok {
		return error.New(
			error.Missing,
			error.Node,
			"cannot uninstall; node not installed",
			out.Pair("node", nodeid))
	}

	delete(n.nodes, nodeid)
	return n.registry.Remove(node.registry.ID)
}

func (n *Nodes) Spawn(nodeid string) *error.Error {
	nodeid = n.resolveId(nodeid)

	n.mu.RLock()
	defer n.mu.RUnlock()

	node, ok := n.nodes[nodeid]
	if !ok {
		return error.New(
			error.Missing,
			error.Node,
			"cannot spawn; node not installed")
	}

	cmd := exec.Command(node.registry.Binary)
	err := cmd.Start()
	if err != nil {
		return error.New(
			error.Generic,
			error.Node,
			"failed to spawn node",
			out.Pair("node", nodeid),
			out.Pair("error", err.Error()))
	}

	return nil
}

func (n *Nodes) Kill(nodeid string) *error.Error {
	nodeid = n.resolveId(nodeid)

	return error.New(
		error.Generic,
		error.Node,
		"not yet implemented")
}

//
//
// private
//

func (n *Nodes) resolveId(nodeid string) string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	_, ok := n.nodes[nodeid]; 
	if ok {
		return nodeid
	}

	suffix := "." + nodeid
	for fullid := range n.nodes {
		if strings.HasSuffix(fullid, suffix) {
			return fullid
		}
	}

	return nodeid
}
