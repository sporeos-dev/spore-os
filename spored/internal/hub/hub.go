// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package hub

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"spored/internal/broadcaster"
	"spored/internal/connection"
	"spored/internal/interfaces"
	"spored/internal/manifest"
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
const hyPhaeNodeID = "dev.sporeos.hyphae"

type Hub struct {
	router interfaces.Router
	registry interfaces.Registry
	witness interfaces.Witness
	broadcaster interfaces.Broadcaster

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
	h.broadcaster = &broadcaster.Broadcaster{}
	h.spore = &spore.Spore{}

	h.router.Open(h, h.spore)
	h.broadcaster.Open(h)
	h.spore.Open(h, h.registry, h.router, h.witness, h.broadcaster)
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
		Broadcaster: h.broadcaster,
	}
	nodeid, err := node.Open(path)
	if err != nil {
		return err
	}
	h.witness.Register(node)

	// Register topics declared in this node's manifest.
	for _, topic := range node.GetManifest().Topics {
		h.broadcaster.AddTopic(topic.Name, nodeid)
	}

	h.nodes[nodeid] = node
	return nil
}

// AddNodeWithManifest loads a node from an already-parsed manifest, skipping
// any direct file reads. Use this when file access was delegated to hyphae.
func (h *Hub) AddNodeWithManifest(m *manifest.Manifest) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	n := &node.Node{
		Router:            h.router,
		WitnessDispatcher: h.witness,
		Broadcaster:       h.broadcaster,
	}
	nodeid, err := n.OpenWithManifest(m)
	if err != nil {
		return err
	}
	h.witness.Register(n)

	for _, topic := range n.GetManifest().Topics {
		h.broadcaster.AddTopic(topic.Name, nodeid)
	}

	h.nodes[nodeid] = n
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

	// PurgeNode cleans up in-flight handles; RemoveRoutes removes command
	// registrations so future callers get RouteNotFound, not RouteNotConnected.
	h.router.PurgeNode(nodeid)
	h.router.RemoveRoutes(nodeid)
	h.broadcaster.PurgeNode(nodeid)
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
		go h.handleConnection(conn)
	}
}

// handleConnection runs in its own goroutine for each inbound socket connection.
// It completes the handshake, verifies the manifest (if not already confirmed at
// startup) and the binary checksum, then hands the connection off to the node.
func (h *Hub) handleConnection(conn net.Conn) {
	isKnown := func(id string) bool {
		h.mu.RLock()
		defer h.mu.RUnlock()
		_, has := h.nodes[id]
		return has
	}

	id, err := shakeHands(conn, isKnown)
	if err != nil {
		slog.Warn("Unable to complete handshake", "error", err)
		conn.Close()
		return
	}

	if err := h.verifyManifest(id); err != nil {
		slog.Warn("Manifest verification failed, rejecting connection", "node", id, "error", err)
		h.witness.Spore(message.SporeEvent("error", "Manifest verification failed", "node="+id, fmt.Sprintf(`error="%s"`, err.Error())))
		rejectAfterHandshake(conn, message.ErrorCodeConnectionFailure, err.Error())
		return
	}

	pid := connection.PeerPID(conn)

	if err := h.verifyBinary(id, pid); err != nil {
		slog.Warn("Binary verification failed, rejecting connection", "node", id, "error", err)
		h.witness.Spore(message.SporeEvent("error", "Binary verification failed", "node="+id, fmt.Sprintf(`error="%s"`, err.Error())))
		rejectAfterHandshake(conn, message.ErrorCodeConnectionFailure, err.Error())
		return
	}

	h.mu.RLock()
	n, has := h.nodes[id]
	h.mu.RUnlock()
	if !has {
		// Safety fallback — shakeHands should have caught this.
		slog.Warn("Unable to find node after handshake", "node", id)
		conn.Close()
		return
	}

	if err := n.Connect(conn); err != nil {
		slog.Warn("Unable to connect", "node", id, "error", err)
		conn.Close()
	}
}

// verifyManifest checks the manifest for nodeid against its stored checksum.
// If the manifest was verified at startup (direct read succeeded), this is a
// no-op. If not — e.g. user-space file not readable by the hub daemon — it
// delegates to HYPHAE.file.hash and compares. Hard-rejects on any failure.
func (h *Hub) verifyManifest(nodeid string) error {
	h.mu.RLock()
	n, has := h.nodes[nodeid]
	h.mu.RUnlock()
	if !has {
		return fmt.Errorf("verifyManifest: node not found: %s", nodeid)
	}

	m := n.GetManifest()
	if h.registry.ManifestVerified(m.Path) {
		return nil // confirmed at startup
	}

	expectedChecksum, ok := h.registry.ManifestChecksumFor(m.Path)
	if !ok {
		return fmt.Errorf("manifest verification: no stored checksum for node %s", nodeid)
	}

	hypha, hyErr := h.GetNode(hyPhaeNodeID)
	if hyErr != nil || !hypha.IsConnected() {
		return fmt.Errorf("manifest not startup-verified and hyphae is not connected for node %s", nodeid)
	}

	reply, err := h.router.RequestNode(hyPhaeNodeID, "HYPHAE.file.hash",
		map[string]string{"path": m.Path})
	if err != nil {
		return fmt.Errorf("manifest verification for %s: HYPHAE.file.hash failed: %w", nodeid, err)
	}

	cap, ok := reply.(*message.Capture)
	if !ok {
		return fmt.Errorf("manifest verification for %s: unexpected reply type from HYPHAE.file.hash", nodeid)
	}
	hash, ok := cap.GetArg("hash")
	if !ok {
		return fmt.Errorf("manifest verification for %s: HYPHAE.file.hash returned no hash", nodeid)
	}

	got := "sha256:" + hash
	if got != expectedChecksum {
		return fmt.Errorf("manifest tampered for %s: expected %s got %s", nodeid, expectedChecksum, got)
	}
	return nil
}

// verifyBinary checks the binary for nodeid against its stored checksum.
// Tries a direct file read first; falls back to HYPHAE.file.hash (using the
// stored binary path) for binaries the hub daemon cannot read directly.
// Hard-rejects on any failure.
//
// The stored binary path is always used — not a PID-derived path — so that
// install-time and connect-time hashing are identical in method and input.
func (h *Hub) verifyBinary(nodeid string, pid int) error {
	h.mu.RLock()
	n, has := h.nodes[nodeid]
	h.mu.RUnlock()
	if !has {
		return fmt.Errorf("verifyBinary: node not found: %s", nodeid)
	}

	m := n.GetManifest()
	if m.App == "" || m.App == "n/a" {
		return nil // no binary declared
	}

	binaryPath, expectedChecksum, ok := h.registry.BinaryChecksumFor(m.Path)
	if !ok {
		slog.Warn("No binary checksum in registry, skipping verification", "node", nodeid)
		return nil
	}

	// Try direct hash first (works for system-space files).
	if got, err := checksumFileDirect(binaryPath); err == nil {
		if got != expectedChecksum {
			return fmt.Errorf("binary tampered for %s: expected %s got %s", nodeid, expectedChecksum, got)
		}
		return nil
	}

	// Direct read failed — delegate to hyphae using the stored binary path.
	// This is the same path and method used at install time, ensuring the
	// hashes are always comparable regardless of how the process was launched.
	hypha, hyErr := h.GetNode(hyPhaeNodeID)
	if hyErr != nil || !hypha.IsConnected() {
		return fmt.Errorf("binary verification for %s: direct hash failed and hyphae is not connected", nodeid)
	}

	reply, err := h.router.RequestNode(hyPhaeNodeID, "HYPHAE.binary.hash",
		map[string]string{"pid": fmt.Sprintf("%d", pid)})
	if err != nil {
		return fmt.Errorf("binary verification for %s: HYPHAE.binary.hash failed: %w", nodeid, err)
	}

	cap, ok := reply.(*message.Capture)
	if !ok {
		return fmt.Errorf("binary verification for %s: unexpected reply type from HYPHAE.binary.hash", nodeid)
	}
	hash, ok := cap.GetArg("hash")
	if !ok {
		return fmt.Errorf("binary verification for %s: HYPHAE.binary.hash returned no hash", nodeid)
	}

	got := "sha256:" + hash
	if got != expectedChecksum {
		return fmt.Errorf("binary tampered (via hyphae) for %s: expected %s got %s", nodeid, expectedChecksum, got)
	}
	return nil
}

// checksumFileDirect computes the SHA-256 of the file at path and returns it
// as "sha256:<lowercasehex>". Used for direct (system-space) binary hashing.
func checksumFileDirect(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// rejectAfterHandshake sends a Spore error message to a connection that has
// already completed the OK handshake, then closes it. This gives the
// connecting node a meaningful rejection rather than a bare EOF.
func rejectAfterHandshake(conn net.Conn, code message.ErrorCode, what string) {
	fmt.Fprintf(conn, "error code=%s what=%q capture=SPORE.hub\n", code, what)
	conn.Close()
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

