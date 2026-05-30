// Copyright 2026 mharr
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
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
app: ./test
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
	manifestsDir, _ := ManifestsDir()
	expectedPath := filepath.Join(manifestsDir, "com.example.test.manifest.spore.yaml")
	if len(r.Paths()) != 1 || r.Paths()[0] != expectedPath {
		t.Errorf("expected system path %q to be registered, got: %v", expectedPath, r.Paths())
	}
}

// =============================================================================
// SPEC §8.2: "Relative paths are relative to the manifest directory."
// The stored copy must carry an absolute app: path so it remains correct after
// the copy is moved to the daemon-owned manifests directory.
// =============================================================================

func TestRegistry_Add_ResolvesRelativeAppPath(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	// app: is a bare relative name — should resolve to dir/node-a
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

	manifestsDir, _ := ManifestsDir()
	copyPath := filepath.Join(manifestsDir, "com.example.test.manifest.spore.yaml")
	m, err := loadStoredManifest(t, copyPath)
	if err != nil {
		t.Fatalf("failed to load stored copy: %v", err)
	}

	want := filepath.Join(dir, "node-a")
	if m.App != want {
		t.Errorf("app: in stored copy: got %q, want %q", m.App, want)
	}
}

func TestRegistry_Add_PreservesAbsoluteAppPath(t *testing.T) {
	r := openRegistry(t)
	dir := t.TempDir()
	absApp := "/usr/local/bin/myapp"
	yaml := `id: com.example.test
name: Test
description: A test node.
schema: SPORE/v1d0
version: 1.0.0
app: ` + absApp + `
api: []
`
	path := writeTempManifest(t, dir, "node.manifest.spore.yaml", yaml)

	if err := r.Add(path); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	manifestsDir, _ := ManifestsDir()
	copyPath := filepath.Join(manifestsDir, "com.example.test.manifest.spore.yaml")
	m, err := loadStoredManifest(t, copyPath)
	if err != nil {
		t.Fatalf("failed to load stored copy: %v", err)
	}

	if m.App != absApp {
		t.Errorf("app: in stored copy: got %q, want %q", m.App, absApp)
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

	manifestsDir, _ := ManifestsDir()
	copyPath := filepath.Join(manifestsDir, "com.example.test.manifest.spore.yaml")
	m, err := loadStoredManifest(t, copyPath)
	if err != nil {
		t.Fatalf("failed to load stored copy: %v", err)
	}

	if m.App != "n/a" {
		t.Errorf("app: in stored copy: got %q, want \"n/a\"", m.App)
	}
}

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

