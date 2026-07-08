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

// manifestWithBinary returns a manifest YAML referencing the given binary path.
func manifestWithBinary(binaryPath string) string {
	return `id: com.example.test
name: Test
description: A test node.
schema: SPORE/v1d0
version: 1.0.0
app: ` + binaryPath + `
api: []
`
}

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
	paths := r.Paths()
	if len(paths) != 1 || paths[0] != path {
		t.Errorf("expected [%q] to be registered, got: %v", path, paths)
	}
}

func TestRegistry_Add_Idempotent(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", validManifestYAML)

	if err := r.Add(path); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if err := r.Add(path); err != nil {
		t.Fatalf("second Add (idempotent): %v", err)
	}
	if len(r.Paths()) != 1 {
		t.Errorf("expected exactly one entry after two adds, got %d", len(r.Paths()))
	}
}

// =============================================================================
// Remove tests
// =============================================================================

func TestRegistry_Remove_Valid(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", validManifestYAML)

	if err := r.Add(path); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := r.Remove(path); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(r.Paths()) != 0 {
		t.Errorf("expected empty registry after remove, got %v", r.Paths())
	}
}

func TestRegistry_Remove_NotRegistered(t *testing.T) {
	r := openRegistry(t)
	err := r.Remove("/some/path/node.manifest.spore.yaml")
	if err == nil {
		t.Fatal("expected error removing unregistered path")
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Errorf("error should mention 'not registered', got: %v", err)
	}
}

// =============================================================================
// Open tests — checksum verification
// =============================================================================

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

func checksumOf(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func TestRegistry_Open_MissingFile(t *testing.T) {
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
	dir := t.TempDir()
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", validManifestYAML)
	checksum := checksumOf(validManifestYAML)
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
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	dir := t.TempDir()
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", validManifestYAML)
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
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	ghostPath := filepath.Join(t.TempDir(), "ghost.manifest.spore.yaml")
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
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	dir := t.TempDir()
	goodPath := writeTempManifest(t, dir, "good.manifest.spore.yaml", validManifestYAML)
	badPath := writeTempManifest(t, dir, "bad.manifest.spore.yaml", validManifestYAML)
	writeRegistryYAML(t, registryFile{
		Version: 1,
		Nodes: []registryEntry{
			{Name: "Good", Manifest: goodPath, Checksum: checksumOf(validManifestYAML)},
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
// Binary checksum tests
// =============================================================================

func writeTempBinary(t *testing.T, dir, filename, content string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(content), 0755); err != nil {
		t.Fatalf("failed to write temp binary: %v", err)
	}
	return path
}

func TestRegistry_Add_WithBinary(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	binaryPath := writeTempBinary(t, dir, "myapp", "fake binary content")
	content := manifestWithBinary(binaryPath)
	manifestPath := writeTempManifest(t, dir, "node.manifest.spore.yaml", content)

	if err := r.Add(manifestPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(r.entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(r.entries))
	}
	entry := r.entries[0]
	if entry.Binary != binaryPath {
		t.Errorf("expected Binary %q, got %q", binaryPath, entry.Binary)
	}
	if entry.BinaryChecksum == "" {
		t.Error("expected BinaryChecksum to be set")
	}
	if entry.BinaryChecksum != checksumOf("fake binary content") {
		t.Errorf("BinaryChecksum mismatch: got %q", entry.BinaryChecksum)
	}
}

func TestRegistry_Add_BinaryNotFound(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	content := manifestWithBinary(filepath.Join(dir, "missing-binary"))
	manifestPath := writeTempManifest(t, dir, "node.manifest.spore.yaml", content)

	err := r.Add(manifestPath)
	if err == nil {
		t.Fatal("expected error when binary is missing")
	}
	if !strings.Contains(err.Error(), "checksum binary") {
		t.Errorf("error should mention 'checksum binary', got: %v", err)
	}
}

func TestRegistry_Open_BinaryChecksumMismatch(t *testing.T) {
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	dir := t.TempDir()
	binaryPath := writeTempBinary(t, dir, "myapp", "fake binary content")
	content := manifestWithBinary(binaryPath)
	manifestPath := writeTempManifest(t, dir, "node.manifest.spore.yaml", content)
	writeRegistryYAML(t, registryFile{
		Version: 1,
		Nodes: []registryEntry{{
			Name:           "Test",
			Manifest:       manifestPath,
			Checksum:       checksumOf(content),
			Binary:         binaryPath,
			BinaryChecksum: "sha256:000000tampered",
		}},
	})

	r := &Registry{}
	if err := r.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(r.Paths()) != 0 {
		t.Errorf("expected tampered binary entry to be skipped, got paths: %v", r.Paths())
	}
}

func TestRegistry_Open_BinaryMissing(t *testing.T) {
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	dir := t.TempDir()
	ghostBinary := filepath.Join(dir, "ghost-binary")
	content := manifestWithBinary(ghostBinary)
	manifestPath := writeTempManifest(t, dir, "node.manifest.spore.yaml", content)
	writeRegistryYAML(t, registryFile{
		Version: 1,
		Nodes: []registryEntry{{
			Name:           "Test",
			Manifest:       manifestPath,
			Checksum:       checksumOf(content),
			Binary:         ghostBinary,
			BinaryChecksum: "sha256:abc",
		}},
	})

	r := &Registry{}
	if err := r.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(r.Paths()) != 0 {
		t.Errorf("expected missing binary entry to be skipped, got paths: %v", r.Paths())
	}
}

func TestRegistry_Open_BinaryValid(t *testing.T) {
	t.Setenv("SPORE_DATA_DIR", t.TempDir())
	dir := t.TempDir()
	binaryPath := writeTempBinary(t, dir, "myapp", "fake binary content")
	content := manifestWithBinary(binaryPath)
	manifestPath := writeTempManifest(t, dir, "node.manifest.spore.yaml", content)
	writeRegistryYAML(t, registryFile{
		Version: 1,
		Nodes: []registryEntry{{
			Name:           "Test",
			Manifest:       manifestPath,
			Checksum:       checksumOf(content),
			Binary:         binaryPath,
			BinaryChecksum: checksumOf("fake binary content"),
		}},
	})

	r := &Registry{}
	if err := r.Open(); err != nil {
		t.Fatalf("Open: %v", err)
	}
	if len(r.Paths()) != 1 || r.Paths()[0] != manifestPath {
		t.Errorf("expected entry to load, got paths: %v", r.Paths())
	}
}

