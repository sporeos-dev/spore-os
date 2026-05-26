package spore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"spored/internal/interfaces"
	"spored/internal/manifest"
	"spored/internal/message"
	"strings"
	"testing"
)

// =============================================================================
// SPEC §10: Hub Manifest — SPORE.* Command Execution
// "The hub exposes subjects for introspection and lifecycle management."
// =============================================================================

// --- Mock types ---

type mockHub struct {
	nodes map[string]*mockNode
}

func (h *mockHub) ListNodes() []string {
	res := make([]string, 0, len(h.nodes))
	for id := range h.nodes {
		res = append(res, id)
	}
	return res
}

func (h *mockHub) GetNode(nodeid string) (interfaces.Node, error) {
	n, has := h.nodes[nodeid]
	if !has {
		return nil, errors.New("missing node " + nodeid)
	}
	return n, nil
}

func (h *mockHub) AddNode(path string) error {
	m, err := manifest.LoadManifest(path)
	if err != nil {
		return err
	}
	h.nodes[m.ID] = &mockNode{id: m.ID, manifest: m}
	return nil
}

func (h *mockHub) RemoveNode(nodeid string) error {
	delete(h.nodes, nodeid)
	return nil
}

func (h *mockHub) SpawnNode(nodeid string) error { return nil }
func (h *mockHub) KillNode(nodeid string) error  { return nil }

type mockNode struct {
	id          string
	manifest    *manifest.Manifest
	sent        []message.Message
	isConnected bool
}

func (n *mockNode) Send(msg message.Message) error {
	n.sent = append(n.sent, msg)
	return nil
}

func (n *mockNode) GetManifest() *manifest.Manifest { return n.manifest }
func (n *mockNode) IsConnected() bool               { return n.isConnected }
func (n *mockNode) GetPID() int                     { return 0 }
func (n *mockNode) SendWitness(msg string)          {}
func (n *mockNode) WitnessNode(msg string)          {}

type mockRegistry struct {
	paths []string
}

func (r *mockRegistry) Open() error         { return nil }
func (r *mockRegistry) Paths() []string     { return r.paths }
func (r *mockRegistry) Add(path string) error {
	r.paths = append(r.paths, path)
	return nil
}
func (r *mockRegistry) Remove(path string) error {
	newPaths := make([]string, 0)
	for _, p := range r.paths {
		if p != path {
			newPaths = append(newPaths, p)
		}
	}
	r.paths = newPaths
	return nil
}

type mockRouter struct {
	commands map[string]string
}

func (r *mockRouter) Open(hub interfaces.Hub, spore interfaces.Spore) {}
func (r *mockRouter) AddRoute(command string, nodeid string) {
	r.commands[command] = nodeid
}
func (r *mockRouter) Route(msg message.Message) error { return nil }
func (r *mockRouter) GetRoute(command string) (string, error) {
	nodeid, has := r.commands[command]
	if !has {
		return "", errors.New("route not found: " + command)
	}
	return nodeid, nil
}
func (r *mockRouter) ListCommands(node string) []string {
	if node == "n/a" || node == "" {
		res := make([]string, 0, len(r.commands))
		for cmd := range r.commands {
			res = append(res, cmd)
		}
		return res
	}
	res := make([]string, 0)
	for cmd, nid := range r.commands {
		if nid == node {
			res = append(res, cmd)
		}
	}
	return res
}
func (r *mockRouter) PurgeNode(nodeID string) {}

type mockWitness struct{}

func (w *mockWitness) Register(node interfaces.Node)   {}
func (w *mockWitness) Unregister(id string)            {}
func (w *mockWitness) Incoming(msg string)             {}
func (w *mockWitness) Outgoing(msg string)             {}
func (w *mockWitness) Spore(msg string)                {}
func (w *mockWitness) Node(msg string, cast string)    {}
// --- Helpers ---

func setupSpore(t *testing.T) (*Spore, *mockHub, *mockRegistry, *mockRouter) {
	t.Helper()

	hub := &mockHub{nodes: make(map[string]*mockNode)}
	reg := &mockRegistry{}
	rtr := &mockRouter{commands: make(map[string]string)}

	s := &Spore{}
	s.hub = hub
	s.registry = reg
	s.router = rtr
	s.witness = &mockWitness{}

	// Load the actual hub manifest
	s.manifest, _ = manifest.LoadManifest("../../spored.manifest.spore.yaml")
	if s.manifest == nil {
		t.Fatal("could not load hub manifest — run tests from spored/")
	}

	return s, hub, reg, rtr
}

func parseSporeMsg(t *testing.T, raw string) *message.Spore {
	t.Helper()
	msg, err := message.Parse(raw, "com.test.caller")
	if err != nil {
		t.Fatalf("failed to parse %q: %v", raw, err)
	}
	sporeMsg, ok := msg.(*message.Spore)
	if !ok {
		t.Fatalf("expected Spore message, got %T", msg)
	}
	return sporeMsg
}

func writeTempManifest(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.manifest.spore.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp manifest: %v", err)
	}
	return path
}

// =============================================================================
// Tests
// =============================================================================

func TestSpore_Help(t *testing.T) {
	s, _, _, _ := setupSpore(t)

	in := parseSporeMsg(t, "SPORE.help ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "ok") {
		t.Error("response should contain ok")
	}
	if !strings.Contains(raw, "capture=dev.sporeos.SPORE") {
		t.Error("response should contain capture=dev.sporeos.SPORE")
	}
	if !strings.Contains(raw, "nodeinfo=") {
		t.Error("response should contain nodeinfo=")
	}
}

func TestSpore_NodeList(t *testing.T) {
	s, hub, _, _ := setupSpore(t)

	// Add a connected and a disconnected node.
	hub.nodes["com.example.clock"] = &mockNode{
		id:          "com.example.clock",
		manifest:    &manifest.Manifest{ID: "com.example.clock"},
		isConnected: true,
	}
	hub.nodes["com.example.timer"] = &mockNode{
		id:          "com.example.timer",
		manifest:    &manifest.Manifest{ID: "com.example.timer"},
		isConnected: false,
	}

	t.Run("without connected flag returns plain IDs", func(t *testing.T) {
		in := parseSporeMsg(t, "SPORE.node.list ~h1")
		out, err := s.Command(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		raw := out.ToString()
		if !strings.Contains(raw, "ok") {
			t.Error("response should contain ok")
		}
		if !strings.Contains(raw, "nodes=") {
			t.Error("response should contain nodes=")
		}
		if !strings.Contains(raw, "com.example.clock") {
			t.Error("should contain com.example.clock")
		}
		if !strings.Contains(raw, "dev.sporeos.SPORE") {
			t.Error("should contain dev.sporeos.SPORE")
		}
		// Must NOT have connected annotations.
		if strings.Contains(raw, " connected") {
			t.Error("plain list should not contain connected token")
		}
	})

	t.Run("with connected flag annotates live nodes only", func(t *testing.T) {
		in := parseSporeMsg(t, "SPORE.node.list connected ~h2")
		out, err := s.Command(in)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		raw := out.ToString()
		if !strings.Contains(raw, "ok") {
			t.Error("response should contain ok")
		}
		if !strings.Contains(raw, "com.example.clock connected") {
			t.Errorf("expected connected token for clock, got: %s", raw)
		}
		// Installed node must appear but WITHOUT the connected token.
		if !strings.Contains(raw, "com.example.timer") {
			t.Errorf("expected timer to appear in list, got: %s", raw)
		}
		if strings.Contains(raw, "com.example.timer connected") {
			t.Errorf("installed node should not have connected token, got: %s", raw)
		}
		if !strings.Contains(raw, "dev.sporeos.SPORE connected") {
			t.Errorf("expected SPORE to be connected, got: %s", raw)
		}
	})
}

func TestSpore_NodeInstall(t *testing.T) {
	s, hub, reg, _ := setupSpore(t)

	manifestYaml := `
id: com.example.test
name: Test Node
description: A test node.
schema: SPORE/v1d0
app: ./test
api:
  - name: test.do
    description: Does something.
    usage:
      - test.do
`
	path := writeTempManifest(t, manifestYaml)

	in := parseSporeMsg(t, "SPORE.node.install path="+path+" ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "ok") {
		t.Errorf("response should contain ok, got: %s", raw)
	}

	// Verify the node was added to the hub
	if _, exists := hub.nodes["com.example.test"]; !exists {
		t.Error("node should have been added to hub")
	}

	// Verify the path was added to registry
	if len(reg.paths) != 1 || reg.paths[0] != path {
		t.Errorf("expected registry to contain path %q, got %v", path, reg.paths)
	}
}

func TestSpore_NodeUninstall_ByNode(t *testing.T) {
	s, hub, reg, _ := setupSpore(t)

	// Pre-install a node
	m := &manifest.Manifest{ID: "com.example.old", Path: "/tmp/old.yaml"}
	hub.nodes["com.example.old"] = &mockNode{id: "com.example.old", manifest: m}
	reg.paths = []string{"/tmp/old.yaml"}

	in := parseSporeMsg(t, "SPORE.node.uninstall node=com.example.old ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out.ToString(), "ok") {
		t.Error("response should contain ok")
	}

	// Node should be removed from hub
	if _, exists := hub.nodes["com.example.old"]; exists {
		t.Error("node should have been removed from hub")
	}

	// Path should be removed from registry
	if len(reg.paths) != 0 {
		t.Errorf("expected empty registry, got %v", reg.paths)
	}
}

func TestSpore_NodeHelp(t *testing.T) {
	s, hub, _, _ := setupSpore(t)

	m := &manifest.Manifest{
		ID:          "com.example.clock",
		Name:        "Clock",
		Description: "Time functions.",
		Version:     "1.0.0",
		Schema:      "SPORE/v1d0",
		App:         "./clock",
		Path:        "/etc/spore/clock.manifest.spore.yaml",
		Api: []manifest.Command{
			{Name: "get_time", Description: "Returns the current time."},
		},
	}
	hub.nodes["com.example.clock"] = &mockNode{id: "com.example.clock", manifest: m}

	in := parseSporeMsg(t, "SPORE.node.help node=com.example.clock ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "ok") {
		t.Error("response should contain ok")
	}
	if !strings.Contains(raw, "nodeinfo=") {
		t.Error("response should contain nodeinfo=")
	}
	if !strings.Contains(raw, "clock.manifest.spore.yaml") {
		t.Error("nodeinfo should contain the manifest path")
	}
}

func TestSpore_NodeHelp_SporeItself(t *testing.T) {
	s, _, _, _ := setupSpore(t)

	in := parseSporeMsg(t, "SPORE.node.help node=dev.sporeos.SPORE ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "nodeinfo=") {
		t.Error("response should contain nodeinfo for SPORE itself")
	}
}

func TestSpore_CommandList(t *testing.T) {
	s, _, _, rtr := setupSpore(t)

	rtr.commands["clock.get_time"] = "com.example.clock"
	rtr.commands["filesystem.read"] = "com.example.filesystem"

	in := parseSporeMsg(t, "SPORE.command.list ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "commands=") {
		t.Error("response should contain commands=")
	}
	// Should include both registered commands and SPORE commands
	if !strings.Contains(raw, "clock.get_time") {
		t.Error("should list clock.get_time")
	}
}

func TestSpore_CommandList_FilteredByNode(t *testing.T) {
	s, _, _, rtr := setupSpore(t)

	rtr.commands["clock.get_time"] = "com.example.clock"
	rtr.commands["filesystem.read"] = "com.example.filesystem"

	in := parseSporeMsg(t, "SPORE.command.list node=com.example.clock ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	// Should only include clock commands, not filesystem
	if strings.Contains(raw, "filesystem.read") {
		t.Error("should not list filesystem.read when filtered by clock node")
	}
}

func TestSpore_CommandHelp(t *testing.T) {
	s, hub, _, rtr := setupSpore(t)

	inputs := []manifest.Input{{Name: "timezone", Type: "string", Description: "IANA timezone.", Required: true}}
	outputs := []manifest.Output{{Name: "time", Type: "string", Description: "Current time."}}
	m := &manifest.Manifest{
		ID:   "com.example.clock",
		Name: "Clock",
		Api: []manifest.Command{
			{
				Name:        "get_time",
				Description: "Returns the current time.",
				Usage:       []string{"clock.get_time", "clock.get_time timezone=UTC"},
				Inputs:      &inputs,
				Outputs:     &outputs,
			},
		},
	}
	hub.nodes["com.example.clock"] = &mockNode{id: "com.example.clock", manifest: m}
	rtr.commands["get_time"] = "com.example.clock"

	in := parseSporeMsg(t, "SPORE.command.help command=get_time ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "commandinfo=") {
		t.Error("response should contain commandinfo=")
	}
}

func TestSpore_CommandHelp_SporeCommand(t *testing.T) {
	s, _, _, _ := setupSpore(t)

	// Ask for help on a SPORE command directly
	in := parseSporeMsg(t, "SPORE.command.help command=SPORE.command.list ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "commandinfo=") {
		t.Error("response should contain commandinfo=")
	}
}

func TestSpore_ErrorList(t *testing.T) {
	s, _, _, _ := setupSpore(t)

	in := parseSporeMsg(t, "SPORE.error.list ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "errors=") {
		t.Error("response should contain errors=")
	}
	// Should contain standard error codes
	if !strings.Contains(raw, "RouteNotFound") {
		t.Error("should list RouteNotFound standard error")
	}
}

func TestSpore_ErrorList_FilteredByNode(t *testing.T) {
	s, hub, _, _ := setupSpore(t)

	m := &manifest.Manifest{
		ID: "com.example.clock",
		Errors: []manifest.ManifestError{
			{Name: "clock.err.invalid_timezone", Description: "Bad timezone."},
		},
	}
	hub.nodes["com.example.clock"] = &mockNode{id: "com.example.clock", manifest: m}

	in := parseSporeMsg(t, "SPORE.error.list node=com.example.clock ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "clock.err.invalid_timezone") {
		t.Error("should list custom error for the specified node")
	}
}

func TestSpore_ErrorHelp_StandardError(t *testing.T) {
	s, _, _, _ := setupSpore(t)

	in := parseSporeMsg(t, "SPORE.error.help error=RouteNotFound ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "errorinfo=") {
		t.Error("response should contain errorinfo=")
	}
	if !strings.Contains(raw, "RouteNotFound") {
		t.Error("should contain error name")
	}
}

func TestSpore_ErrorHelp_CustomError(t *testing.T) {
	s, hub, _, _ := setupSpore(t)

	m := &manifest.Manifest{
		ID: "com.example.clock",
		Errors: []manifest.ManifestError{
			{Name: "clock.err.invalid_timezone", Description: "Bad timezone.", Examples: []string{"Misspelled"}},
		},
	}
	hub.nodes["com.example.clock"] = &mockNode{id: "com.example.clock", manifest: m}

	in := parseSporeMsg(t, "SPORE.error.help error=clock.err.invalid_timezone ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()
	if !strings.Contains(raw, "errorinfo=") {
		t.Error("response should contain errorinfo=")
	}
}

func TestSpore_ErrorHelp_NotFound(t *testing.T) {
	s, _, _, _ := setupSpore(t)

	in := parseSporeMsg(t, "SPORE.error.help error=NonexistentCode ~h1")
	_, err := s.Command(in)
	if err == nil {
		t.Error("expected error for nonexistent error code")
	}
}

func TestSpore_UnknownCommand_ReturnsError(t *testing.T) {
	s, _, _, _ := setupSpore(t)

	in := parseSporeMsg(t, "SPORE.nonexistent ~h1")
	_, err := s.Command(in)
	if err == nil {
		t.Error("expected error for unknown SPORE command")
	}
}

func TestSpore_ResponseFormat_CaptureAndOk(t *testing.T) {
	// Every SPORE command response should have ok and capture=dev.sporeos.SPORE
	s, _, _, _ := setupSpore(t)

	commands := []string{
		"SPORE.help ~h1",
		"SPORE.node.list ~h2",
		"SPORE.error.list ~h3",
	}

	for _, cmd := range commands {
		in := parseSporeMsg(t, cmd)
		out, err := s.Command(in)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", cmd, err)
		}

		raw := out.ToString()
		if !out.IsCapture() {
			t.Errorf("%q: response should be a capture, got type flags: capture=%v", cmd, out.IsCapture())
		}
		if !strings.Contains(raw, "capture=dev.sporeos.SPORE") {
			t.Errorf("%q: should contain capture=dev.sporeos.SPORE, got: %s", cmd, raw)
		}
	}
}

func TestSpore_NodeList_IncludesSpore(t *testing.T) {
	// SPEC §10: SPORE.node.list lists all known nodes.
	// The SPORE hub itself should appear in the list.
	s, _, _, _ := setupSpore(t)

	in := parseSporeMsg(t, "SPORE.node.list ~h1")
	out, err := s.Command(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := out.ToString()

	// Extract the nodes= JSON value
	idx := strings.Index(raw, "nodes=")
	if idx == -1 {
		t.Fatal("response missing nodes=")
	}
	nodesStr := raw[idx+len("nodes="):]
	// The JSON array may have spaces after it — find the closing bracket
	end := strings.Index(nodesStr, "]")
	if end == -1 {
		t.Fatal("malformed nodes array")
	}
	nodesStr = nodesStr[:end+1]

	var nodes []string
	if err := json.Unmarshal([]byte(nodesStr), &nodes); err != nil {
		t.Fatalf("failed to parse nodes JSON: %v (raw: %q)", err, nodesStr)
	}

	found := false
	for _, n := range nodes {
		if n == "dev.sporeos.SPORE" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("dev.sporeos.SPORE not in node list: %v", nodes)
	}
}
