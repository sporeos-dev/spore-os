// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"spored/internal/manifest"
	"spored/internal/utilities"
	"strings"

	"gopkg.in/yaml.v3"
)

// registryFile is the on-disk structure of nodes.registry.yaml.
type registryFile struct {
	Version int             `yaml:"version"`
	Nodes   []registryEntry `yaml:"nodes"`
}

// registryEntry is one installed node in nodes.registry.yaml.
type registryEntry struct {
	Name     string `yaml:"name"`
	Manifest string `yaml:"manifest"`
	Checksum string `yaml:"checksum"`
}

type Registry struct {
	entries []registryEntry
}

// Open reads nodes.registry.yaml, re-hashes every manifest at the recorded
// path, and only keeps entries whose checksum matches. A missing registry file
// is not an error — it means no nodes have been installed yet.
func (r *Registry) Open() error {
	registryPath, err := RegistryPath()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(registryPath)
	if os.IsNotExist(err) {
		r.entries = nil
		return nil
	}
	if err != nil {
		return fmt.Errorf("registry: read nodes.registry.yaml: %w", err)
	}

	var rf registryFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return fmt.Errorf("registry: parse nodes.registry.yaml: %w", err)
	}

	r.entries = nil
	for _, entry := range rf.Nodes {
		got, err := checksumFile(entry.Manifest)
		if err != nil {
			slog.Warn("Registry: cannot read manifest, skipping node",
				"name", entry.Name, "manifest", entry.Manifest, "error", err)
			continue
		}
		if got != entry.Checksum {
			slog.Warn("Registry: checksum mismatch, refusing to load node",
				"name", entry.Name, "manifest", entry.Manifest,
				"expected", entry.Checksum, "got", got)
			continue
		}
		r.entries = append(r.entries, entry)
	}

	return nil
}

// Paths returns the manifest file paths for all verified entries.
func (r *Registry) Paths() []string {
	paths := make([]string, 0, len(r.entries))
	for _, e := range r.entries {
		paths = append(paths, e.Manifest)
	}
	return paths
}

// Add installs a node manifest. It validates the manifest, resolves any
// relative app: path, copies the file into store/<id>/<id>.manifest.spore.yaml,
// computes a SHA-256 checksum of the stored copy, then writes the entry into
// nodes.registry.yaml. Calling Add again for the same node ID overwrites the
// existing entry (useful for upgrades).
func (r *Registry) Add(path string) error {
	if !strings.HasSuffix(path, ".manifest.spore.yaml") {
		return fmt.Errorf("registry: %q is not a .manifest.spore.yaml file", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("registry: manifest file not found: %q", path)
	}

	m, err := manifest.LoadManifest(path)
	if err != nil {
		return fmt.Errorf("registry: invalid manifest %q: %w", path, err)
	}

	// Resolve app: relative to the source manifest's directory (SPEC §8.2)
	// before we copy anything — the source directory won't be relevant once
	// everything is in the store.
	resolvedApp, err := utilities.ResolveAppPath(m.App, filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("registry: resolve app path: %w", err)
	}
	m.App = resolvedApp

	// Create store/<id>/ to hold both the binary and the manifest.
	storeDir, err := StoreDir()
	if err != nil {
		return fmt.Errorf("registry: store dir: %w", err)
	}
	nodeDir := filepath.Join(storeDir, m.ID)
	if err := os.MkdirAll(nodeDir, 0755); err != nil {
		return fmt.Errorf("registry: create node dir: %w", err)
	}

	// Copy the app binary into store/<id>/<binary-name> and update app: to the
	// store path. Skip for nodes that declare app: n/a (daemon-only / virtual).
	if m.App != "n/a" {
		appDest := filepath.Join(nodeDir, filepath.Base(m.App))
		if err := copyFile(m.App, appDest, 0755); err != nil {
			return fmt.Errorf("registry: copy app binary: %w", err)
		}
		m.App = appDest
	}

	// Write the manifest (with the updated app: path) into store/<id>/.
	systemPath := filepath.Join(nodeDir, m.ID+".manifest.spore.yaml")
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("registry: serialise manifest: %w", err)
	}
	if err := os.WriteFile(systemPath, data, 0644); err != nil {
		return fmt.Errorf("registry: write manifest copy: %w", err)
	}

	// Compute the checksum of what we actually wrote.
	checksum, err := checksumFile(systemPath)
	if err != nil {
		return fmt.Errorf("registry: compute checksum: %w", err)
	}

	// Add or replace the entry for this node.
	entry := registryEntry{Name: m.Name, Manifest: systemPath, Checksum: checksum}
	replaced := false
	for i, e := range r.entries {
		if e.Manifest == systemPath {
			r.entries[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		r.entries = append(r.entries, entry)
	}

	return r.save()
}

// Remove removes the entry with the given manifest path from the registry and
// updates nodes.registry.yaml. If the manifest lives inside the store directory
// it is also deleted from disk.
func (r *Registry) Remove(manifestPath string) error {
	var remaining []registryEntry
	found := false
	for _, e := range r.entries {
		if e.Manifest == manifestPath {
			found = true
		} else {
			remaining = append(remaining, e)
		}
	}
	if !found {
		return fmt.Errorf("registry: manifest not registered: %q", manifestPath)
	}
	r.entries = remaining

	if err := r.save(); err != nil {
		return err
	}

	// Clean up the manifest file and its per-node directory from store/.
	if storeDir, err := StoreDir(); err == nil && strings.HasPrefix(manifestPath, storeDir) {
		_ = os.Remove(manifestPath)
		_ = os.Remove(filepath.Dir(manifestPath))
	}
	return nil
}

func (r *Registry) save() error {
	registryPath, err := RegistryPath()
	if err != nil {
		return err
	}
	if err := EnsureRegistryDir(registryPath); err != nil {
		return fmt.Errorf("registry: ensure dir: %w", err)
	}
	nodes := r.entries
	if nodes == nil {
		nodes = []registryEntry{}
	}
	rf := registryFile{Version: 1, Nodes: nodes}
	data, err := yaml.Marshal(rf)
	if err != nil {
		return fmt.Errorf("registry: marshal: %w", err)
	}
	return os.WriteFile(registryPath, data, 0600)
}

// checksumFile computes the SHA-256 of the file at path and returns it as
// "sha256:<lowercasehex>".
func checksumFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// copyFile copies the file at src to dst with the given permission bits,
// creating or truncating dst. Used to place app binaries in the store.
func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err = io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
