package nodes

import (
	"bufio"
	"log/slog"
	"net"
	"os/exec"
	"spored/internal/manifest"
	"spored/internal/registry"
	"spored/internal/utilities/build"
	"spored/internal/utilities/file"
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
}

func (n *Nodes) Close() {
	n.registry.Close()
}

func (n *Nodes) Autostart() {
	
}

func (n *Nodes) HandleConnection(conn net.Conn) {
	conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetReadDeadline(time.Time{})

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		slog.Error("Failed to receive data", "error", err)
		writer.WriteString(build.Error(error.New(error.HandshakeDenial, "did not receive node id")))
		writer.Flush()
		conn.Close()
		return
	}

	nodeid := strings.TrimSpace(line)
	n.mu.RLock()
	defer n.mu.RUnlock()
	node, ok := n.nodes[nodeid]
	if !ok {
		slog.Error("Unknown node", "nodeid", nodeid)
		writer.WriteString(build.Error(error.New(error.HandshakeDenial, "node not installed"), "cast", nodeid))
		writer.Flush()
		conn.Close()
		return
	}

	// Handle the connection with the node
	sp_err := node.handleConnection(conn, reader, writer)
	if sp_err != nil {
		writer.WriteString(build.Error(sp_err, "cast", nodeid))
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

	n.mu.RLock()
	defer n.mu.RUnlock()

	node, ok := n.nodes[nodeid]
	if !ok { return nil }
	
	return node.manifest
}

func (n *Nodes) Install(path string) *error.Error {

	if !file.Exists(path) {
		return error.New(error.InstallationFailure, "bad path")
	}

	// readable, handle via spore
	if file.IsReadable(path) {

		manifest := manifest.ManifestFromPath(path)
		if manifest == nil {
			return error.New(error.InstallationFailure, "failed to load manifest")
		}

		registryElement := registry.ElementFromManifest(manifest)
		if registryElement == nil {
			return error.New(error.InstallationFailure, "failed to create registry element")
		}

		node := newNode(registryElement, manifest)
		n.nodes[node.registry.ID] = node

	// not readable, handle via hyphae
	} else {

	}

	return nil
}

func (n *Nodes) Uninstall(nodeid string) *error.Error {
	
	n.mu.Lock()
	defer n.mu.Unlock()

	node, ok := n.nodes[nodeid]
	if !ok {
		return error.New(error.InstallationFailure, "cannot uninstall; node not installed")
	}

	delete(n.nodes, nodeid)
	return n.registry.Remove(node.registry.ID)
}

func (n *Nodes) Spawn(nodeid string) *error.Error {

	n.mu.RLock()
	defer n.mu.RUnlock()

	node, ok := n.nodes[nodeid]
	if !ok {
		return error.New(error.Generic, "cannot spawn; node not installed")
	}

	cmd := exec.Command(node.registry.Binary)
	if err := cmd.Start(); err != nil {
		return error.New(error.Generic, "failed to spawn node", "error", err.Error())
	}

	return nil
}

func (n *Nodes) Kill(nodeid string) *error.Error {
	return nil
	
}
