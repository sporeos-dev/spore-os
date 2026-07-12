package nodes

import (
	"bufio"
	"log/slog"
	"net"
	"spored/internal/utilities/build"
	"strings"
	"sync"
	"time"

	"spored/internal/utilities/error"
)

const handshakeTimeout = 5 * time.Second

type Nodes struct {
	registry *registry

	mu sync.RWMutex
	nodes map[string]inode

	bus ibus
	hyphae ihyphae
	spore ispore
}

func New() *Nodes {
	nodes := &Nodes{
		registry: newRegistry(),
		nodes:   make(map[string]inode),
	}

	nodes.mu.Lock()
	defer nodes.mu.Unlock()

	s := newSpore(nodes.spore)
	nodes.nodes[s.id()] = s

	for _, el := range nodes.registry.Elements {
		n := newNode(el)
		nodes.nodes[n.id()] = n
	}

	return nodes
}

func (n *Nodes) Set(bus ibus, hyphae ihyphae, spore ispore) {
	n.bus = bus
	n.hyphae = hyphae
	n.spore = spore
}

func (n *Nodes) Close() {
	n.registry.close()
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


