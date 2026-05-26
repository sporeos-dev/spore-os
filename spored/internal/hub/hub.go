package hub

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"spored/internal/interfaces"
	"spored/internal/message"
	"spored/internal/node"
	"spored/internal/registry"
	"spored/internal/spore"
	"spored/internal/spore/router"
	"spored/internal/utilities"
	"spored/internal/witness"
	"strings"
	"sync"
	"syscall"
	"time"
)

const handshakeTimeout = 5 * time.Second

type Hub struct {
	router interfaces.Router
	registry interfaces.Registry
	witness interfaces.Witness

	mu sync.RWMutex
	nodes map[string] *node.Node
	spore interfaces.Spore

	listener net.Listener
}

func (h *Hub) Open() error {

	os.Remove(socketPath())

	h.router = &router.Router{}
	h.registry = &registry.Registry{}
	h.witness = &witness.Witness{}
	h.spore = &spore.Spore{}

	h.router.Open(h, h.spore)
	h.spore.Open(h, h.registry, h.router, h.witness)
	h.nodes = make(map[string]*node.Node)

	var err error	
	sockPath := socketPath()
	h.listener, err = net.Listen("unix", sockPath)
	if err != nil {
		return err
	}
	// All users must be able to connect; identity is established via peer
	// credentials captured on each connection (peercred_darwin.go etc.).
	if err := os.Chmod(sockPath, 0777); err != nil {
		return fmt.Errorf("chmod socket: %w", err)
	}

	err = h.registry.Open()
	if err != nil {
		return err
	}
	paths := h.registry.Paths()

	for _, path := range paths {
		err := h.AddNode(path)
		if err != nil {
			slog.Warn("Unable to read registry", "path", path, "error", err)
		}
	}

	var spawnMessages []string
	for _, nodeid := range h.ListNodes() {
		n, err := h.GetNode(nodeid)
		if err != nil {
			continue
		}
		if n.GetManifest().Autostart {
			if err := h.SpawnNode(nodeid); err != nil {
				spawnMessages = append(spawnMessages, message.SporeEvent("warn", "Autostart failed", "node="+nodeid, fmt.Sprintf(`error="%s"`, err.Error())))
				slog.Warn("Autostart failed", "node", nodeid, "error", err)
			} else {
				spawnMessages = append(spawnMessages, message.SporeEvent("info", "Node autostarted", "node="+nodeid))
				slog.Info("Autostarted node", "node", nodeid)
			}
		}
	}

	h.witness.Spore(message.SporeEvent("info", "Spawning SPORE"))
	for _, el := range spawnMessages {
		h.witness.Spore(el)
	}

	go h.run()
	return nil
}

func (h *Hub) Close() error {
	h.witness.Spore(message.SporeEvent("info", "Killing SPORE"))
	h.listener.Close()
	return nil
}

func (h *Hub) ListNodes() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	res := make([]string, 0, len(h.nodes))
	for nodeid := range h.nodes {
		res = append(res, nodeid)
	}

	return res
}

func (h *Hub) GetNode(nodeid string) (interfaces.Node, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	
	node, has := h.nodes[nodeid]
	if !has {
		return nil, errors.New("missing node " + nodeid)
	}
	
	return node, nil
}

func (h *Hub) AddNode(path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	node := &node.Node{
		Router: h.router,
		WitnessDispatcher: h.witness,
	}
	nodeid, err := node.Open(path)
	if err != nil {
		return err
	}
	h.witness.Register(node)
	
	h.nodes[nodeid] = node
	return nil
}

func (h *Hub) RemoveNode(nodeid string) error {
	h.witness.Unregister(nodeid)
	h.mu.Lock()

	node, has := h.nodes[nodeid]
	if !has {
		h.mu.Unlock()
		return errors.New("missing node " + nodeid)
	}
	// Disconnect is best-effort — a dead socket is not a blocker.
	node.Disconnect()
	delete(h.nodes, nodeid)

	h.mu.Unlock()

	// PurgeNode calls hub.GetNode internally, so it must run outside the hub lock.
	h.router.PurgeNode(nodeid)
	return nil
}

func (h *Hub) SpawnNode(nodeid string) error {
	node, err := h.GetNode(nodeid)
	if err != nil {
		return err
	}

	if node.IsConnected() {
		return errors.New("node is already running")
	}

	m := node.GetManifest()
	if m.App == "" || m.App == "n/a" {
		return errors.New("node has no executable defined")
	}

	exe, err := expandPath(m.App, m.Path)
	if err != nil {
		return err
	}

	cmd := exec.Command(exe)
	return cmd.Start()
}

func (h *Hub) KillNode(nodeid string) error {
	node, err := h.GetNode(nodeid)
	if err != nil {
		return err
	}

	if !node.IsConnected() {
		return errors.New("node is not running")
	}

	pid := node.GetPID()
	if pid == 0 {
		return errors.New("peer PID not available on this platform")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return err
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return err
	}

	// Wait up to 3 seconds for graceful exit, then force-kill.
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := process.Signal(syscall.Signal(0)); err != nil {
			// Process is gone — clean exit after SIGTERM.
			return nil
		}
	}

	return process.Signal(syscall.SIGKILL)
}

func (h *Hub) run() {
	for {
		conn, err := h.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			slog.Error("Listener error", "error", err)
			continue
		}

		isKnown := func(id string) bool {
			h.mu.RLock()
			defer h.mu.RUnlock()
			_, has := h.nodes[id]
			return has
		}

		res, err := shakeHands(conn, isKnown)
		if err != nil {
			slog.Warn("Unable to complete handshake", "error", err)
			conn.Close()
			continue
		}

		node, has := h.nodes[res]
		if !has {
			// Safety fallback — shakeHands should have caught this.
			slog.Warn("Unable to find node after handshake", "node", res)
			conn.Close()
			continue
		}

		err = node.Connect(conn)
		if err != nil {
			slog.Warn("Unable to connect", "node", res, "error", err)
			conn.Close()
			continue
		}
	}
}


// expandPath resolves an app path from a manifest.
// Delegates to pathutil.ResolveAppPath; manifestPath is the manifest file path
// (the directory component is used as the base for relative paths).
func expandPath(app, manifestPath string) (string, error) {
	return utilities.ResolveAppPath(app, filepath.Dir(manifestPath))
}

func shakeHands(conn net.Conn, isKnown func(string) bool) (string, error) {
	conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetReadDeadline(time.Time{})

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		writer.WriteString("ERR " + err.Error() + "\n")
		writer.Flush()
		return "", err
	}

	id := strings.TrimSpace(line)
	if !isKnown(id) {
		writer.WriteString(fmt.Sprintf(`error code=RouteNotFound what="Node not installed: %s" capture=SPORE.hub`+"\n", id))
		writer.Flush()
		return "", fmt.Errorf("node not installed: %s", id)
	}

	writer.WriteString("OK\n")
	writer.Flush()

	return id, nil
}

