// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const validManifestYAML = `id: com.example.test
name: Test
description: A test node.
schema: SPORE/v1d0
version: 1.0.0
app: "n/a"
api: []
`

const invalidManifestYAML = `id: SPORE.reserved
name: Bad
`

func writeTempManifest(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp manifest: %v", err)
	}
	return path
}

func openRegistry(t *testing.T) *Registry {
	t.Helper()
	r := &Registry{}
	// Point the registry file to a temp location so tests don't touch the real one.
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	if err := r.Open(); err != nil {
		t.Fatalf("failed to open registry: %v", err)
	}
	return r
}

// =============================================================================
// Add tests
// =============================================================================

func TestRegistry_Add_WrongExtension(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "node.yaml")
	os.WriteFile(path, []byte(validManifestYAML), 0644)

	err := r.Add(path)
	if err == nil {
		t.Fatal("expected error for wrong file extension")
	}
	if !strings.Contains(err.Error(), ".manifest.spore.yaml") {
		t.Errorf("error should mention .manifest.spore.yaml, got: %v", err)
	}
}

func TestRegistry_Add_FileNotFound(t *testing.T) {
	r := openRegistry(t)

	err := r.Add("/nonexistent/path/node.manifest.spore.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should mention 'not found', got: %v", err)
	}
}

func TestRegistry_Add_InvalidManifest(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", invalidManifestYAML)

	err := r.Add(path)
	if err == nil {
		t.Fatal("expected error for invalid manifest YAML")
	}
	if !strings.Contains(err.Error(), "invalid manifest") {
		t.Errorf("error should mention 'invalid manifest', got: %v", err)
	}
}

func TestRegistry_Add_Valid(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", validManifestYAML)

	if err := r.Add(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	storeDir, _ := StoreDir()
	expectedPath := filepath.Join(storeDir, "com.example.test", "com.example.test.manifest.spore.yaml")
	if len(r.Paths()) != 1 || r.Paths()[0] != expectedPath {
		t.Errorf("expected system path %q to be registered, got: %v", expectedPath, r.Paths())
	}
}

// =============================================================================
// SPEC §8.2: "Relative paths are relative to the manifest directory."
// The stored copy must carry an absolute app: path so it remains correct after
// the copy is moved to the daemon-owned store directory.
// =============================================================================

func TestRegistry_Add_ResolvesRelativeAppPath(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	// Create a dummy binary alongside the manifest.
	binaryPath := filepath.Join(dir, "node-a")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatalf("create dummy binary: %v", err)
	}
	yaml := `id: com.example.test
name: Test
description: A test node.
schema: SPORE/v1d0
version: 1.0.0
app: node-a
api: []
`
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", yaml)

	if err := r.Add(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	storeDir, _ := StoreDir()
	copyPath := filepath.Join(storeDir, "com.example.test", "com.example.test.manifest.spore.yaml")
	m, err := loadStoredManifest(t, copyPath)
	if err != nil {
		t.Fatalf("failed to load stored copy: %v", err)
	}

	// app: should now point to the binary copy inside the store.
	want := filepath.Join(storeDir, "com.example.test", "node-a")
	if m.App != want {
		t.Errorf("app: in stored copy: got %q, want %q", m.App, want)
	}
	// The binary itself should exist in the store.
	if _, err := os.Stat(want); err != nil {
		t.Errorf("binary not found in store at %q: %v", want, err)
	}
}

func TestRegistry_Add_CopiesBinaryToStore(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	// Place a dummy binary at an absolute path inside a temp dir.
	binaryPath := filepath.Join(dir, "myapp")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatalf("create dummy binary: %v", err)
	}
	yaml := `id: com.example.test
name: Test
description: A test node.
schema: SPORE/v1d0
version: 1.0.0
app: ` + binaryPath + `
api: []
`
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", yaml)

	if err := r.Add(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	storeDir, _ := StoreDir()
	copyPath := filepath.Join(storeDir, "com.example.test", "com.example.test.manifest.spore.yaml")
	m, err := loadStoredManifest(t, copyPath)
	if err != nil {
		t.Fatalf("failed to load stored copy: %v", err)
	}

	// app: should point to the binary copy inside the store, not the original.
	want := filepath.Join(storeDir, "com.example.test", "myapp")
	if m.App != want {
		t.Errorf("app: in stored copy: got %q, want %q", m.App, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Errorf("binary not found in store at %q: %v", want, err)
	}
}

func TestRegistry_Add_PreservesNA(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	yaml := `id: com.example.test
name: Test
description: A test node.
schema: SPORE/v1d0
version: 1.0.0
app: "n/a"
api: []
`
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", yaml)

	if err := r.Add(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	storeDir, _ := StoreDir()
	copyPath := filepath.Join(storeDir, "com.example.test", "com.example.test.manifest.spore.yaml")
	m, err := loadStoredManifest(t, copyPath)
	if err != nil {
		t.Fatalf("failed to load stored copy: %v", err)
	}

	if m.App != "n/a" {
		t.Errorf("app: in stored copy: got %q, want \"n/a\"", m.App)
	}
}

// =============================================================================
// Open tests — checksum verification
// =============================================================================

// writeRegistryYAML writes a nodes.registry.yaml directly into the data root.
// Used to simulate what the external installer would produce.
func writeRegistryYAML(t *testing.T, rf registryFile) {
	t.Helper()
	registryPath, err := RegistryPath()
	if err != nil {
		t.Fatalf("RegistryPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(registryPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, err := yaml.Marshal(rf)
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	if err := os.WriteFile(registryPath, data, 0644); err != nil {
		t.Fatalf("write nodes.registry.yaml: %v", err)
	}
}

// writeStoreManifest places content at store/<name>/<name>.manifest.spore.yaml
// and returns the path and its sha256:<hex> checksum.
func writeStoreManifest(t *testing.T, name, content string) (path, checksum string) {
	t.Helper()
	storeDir, err := StoreDir()
	if err != nil {
		t.Fatalf("StoreDir: %v", err)
	}
	nodeDir := filepath.Join(storeDir, name)
	if err := os.MkdirAll(nodeDir, 0755); err != nil {
		t.Fatalf("mkdir store/%s: %v", name, err)
	}
	path = filepath.Join(nodeDir, name+".manifest.spore.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	sum := sha256.Sum256([]byte(content))
	checksum = "sha256:" + hex.EncodeToString(sum[:])
	return path, checksum
}

func TestRegistry_Open_MissingFile(t *testing.T) {
	// No nodes.registry.yaml — Open should succeed with an empty registry.
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	r := &Registry{}
	if err := r.Open(); err != nil {
		t.Fatalf("Open with no registry file: %v", err)
	}
	if len(r.Paths()) != 0 {
		t.Errorf("expected no paths, got %v", r.Paths())
	}
}

func TestRegistry_Open_ValidEntry(t *testing.T) {
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	path, checksum := writeStoreManifest(t, "com.example.test", validManifestYAML)
	writeRegistryYAML(t, registryFile{
		Version: 1,
		Nodes:   []registryEntry{{Name: "Test", Manifest: path, Checksum: checksum}},
	})

	r := &Registry{}
	if err := r.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(r.Paths()) != 1 || r.Paths()[0] != path {
		t.Errorf("expected [%q], got %v", path, r.Paths())
	}
}

func TestRegistry_Open_ChecksumMismatch(t *testing.T) {
	// A tampered checksum must cause the entry to be silently skipped.
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	path, _ := writeStoreManifest(t, "com.example.test", validManifestYAML)
	writeRegistryYAML(t, registryFile{
		Version: 1,
		Nodes:   []registryEntry{{Name: "Test", Manifest: path, Checksum: "sha256:000000"}},
	})

	r := &Registry{}
	if err := r.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(r.Paths()) != 0 {
		t.Errorf("expected tampered entry to be skipped, got paths: %v", r.Paths())
	}
}

func TestRegistry_Open_ManifestFileMissing(t *testing.T) {
	// A registry entry whose manifest file has been deleted is silently skipped.
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	storeDir, _ := StoreDir()
	ghostPath := filepath.Join(storeDir, "com.example.ghost", "com.example.ghost.manifest.spore.yaml")
	writeRegistryYAML(t, registryFile{
		Version: 1,
		Nodes:   []registryEntry{{Name: "Ghost", Manifest: ghostPath, Checksum: "sha256:abc"}},
	})

	r := &Registry{}
	if err := r.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(r.Paths()) != 0 {
		t.Errorf("expected missing manifest to be skipped, got paths: %v", r.Paths())
	}
}

func TestRegistry_Open_PartialLoad(t *testing.T) {
	// Mix of a valid and a tampered entry — only the valid one should load.
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	goodPath, goodSum := writeStoreManifest(t, "com.example.good", validManifestYAML)
	badPath, _ := writeStoreManifest(t, "com.example.bad", validManifestYAML)
	writeRegistryYAML(t, registryFile{
		Version: 1,
		Nodes: []registryEntry{
			{Name: "Good", Manifest: goodPath, Checksum: goodSum},
			{Name: "Bad", Manifest: badPath, Checksum: "sha256:tampered"},
		},
	})

	r := &Registry{}
	if err := r.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	paths := r.Paths()
	if len(paths) != 1 || paths[0] != goodPath {
		t.Errorf("expected only good entry, got %v", paths)
	}
}

// =============================================================================
// Helpers
// =============================================================================

// loadStoredManifest reads and unmarshals a stored manifest YAML file.
func loadStoredManifest(t *testing.T, path string) (*storedApp, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s storedApp
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

type storedApp struct {
	App string `yaml:"app"`
}
