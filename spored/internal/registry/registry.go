// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"spored/internal/manifest"
	"strings"

	"gopkg.in/yaml.v3"
)

// registryFile is the on-disk structure of nodes.registry.yaml.
type registryFile struct {
	Version int             `yaml:"version"`
	Nodes   []registryEntry `yaml:"nodes"`
}

// registryEntry is one installed node in nodes.registry.yaml.
// Manifest points to the manifest at its original installed location —
// nothing is copied into the data root.
// Binary is the absolute path to the application binary (empty when app is "n/a").
// verified is in-memory only: true when both the manifest and binary checksums
// were confirmed at startup (direct read). False means the hub lacked read
// access (user-space files) — both are re-verified at connect time via hyphae.
type registryEntry struct {
	Name           string `yaml:"name"`
	Manifest       string `yaml:"manifest"`
	Checksum       string `yaml:"checksum"`
	Binary         string `yaml:"binary"`
	BinaryChecksum string `yaml:"binary_checksum"`
	verified       bool
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
			if os.IsNotExist(err) {
				// Manifest was deleted — drop the entry entirely.
				slog.Warn("Registry: manifest not found, dropping entry",
					"name", entry.Name, "manifest", entry.Manifest)
				continue
			}
			// Hub cannot access this file — likely installed from outside the hub's
			// session (e.g. a user-space path). Keep the entry unverified; both
			// manifest and binary will be verified via hyphae at connect time.
			slog.Info("Registry: manifest not accessible to hub at startup, will verify at connect time",
				"name", entry.Name, "manifest", entry.Manifest)
			slog.Debug("Registry: manifest access error detail", "name", entry.Name, "error", err)
			entry.verified = false
			r.entries = append(r.entries, entry)
			continue
		}
		if got != entry.Checksum {
			slog.Warn("Registry: manifest checksum mismatch, refusing to load node",
				"name", entry.Name, "manifest", entry.Manifest,
				"expected", entry.Checksum, "got", got)
			continue
		}
		if entry.Binary != "" && entry.BinaryChecksum != "" {
			gotBinary, err := checksumFile(entry.Binary)
			if err != nil {
				if os.IsNotExist(err) {
					slog.Warn("Registry: binary not found, dropping entry",
						"name", entry.Name, "binary", entry.Binary)
					continue
				}
				// Hub cannot access this binary — keep entry unverified.
				slog.Info("Registry: binary not accessible to hub at startup, will verify at connect time",
					"name", entry.Name, "binary", entry.Binary)
				slog.Debug("Registry: binary access error detail", "name", entry.Name, "error", err)
				entry.verified = false
				r.entries = append(r.entries, entry)
				continue
			}
			if gotBinary != entry.BinaryChecksum {
				slog.Warn("Registry: binary checksum mismatch, refusing to load node",
					"name", entry.Name, "binary", entry.Binary,
					"expected", entry.BinaryChecksum, "got", gotBinary)
				continue
			}
		}
		entry.verified = true
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

// Add registers a manifest path. It validates the manifest, computes a
// SHA-256 checksum of the file at its current location, and writes the entry
// into nodes.registry.yaml. Nothing is copied — the manifest stays where it is.
// Calling Add again for the same path refreshes the checksum (useful after
// an upgrade).
func (r *Registry) Add(path string) error {
	if !strings.HasSuffix(path, ".manifest.spore.yaml") {
		return fmt.Errorf("registry: %q is not a .manifest.spore.yaml file", path)
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("registry: resolve path %q: %w", path, err)
	}
	path = abs

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("registry: manifest file not found: %q", path)
	}

	m, err := manifest.LoadManifest(path)
	if err != nil {
		return fmt.Errorf("registry: invalid manifest %q: %w", path, err)
	}

	checksum, err := checksumFile(path)
	if err != nil {
		return fmt.Errorf("registry: checksum manifest: %w", err)
	}

	entry := registryEntry{Name: m.Name, Manifest: path, Checksum: checksum}

	if m.App != "n/a" {
		binaryPath := m.App
		if !filepath.IsAbs(binaryPath) {
			binaryPath = filepath.Join(filepath.Dir(path), binaryPath)
		}
		binaryChecksum, err := checksumFile(binaryPath)
		if err != nil {
			return fmt.Errorf("registry: checksum binary %q: %w", binaryPath, err)
		}
		entry.Binary = binaryPath
		entry.BinaryChecksum = binaryChecksum
	}
	replaced := false
	for i, e := range r.entries {
		if e.Manifest == path {
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
// updates nodes.registry.yaml.
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
	return r.save()
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

// AddEntry registers a node using pre-computed checksums — used when the hub
// delegates file reads to spore-hyphae (user-space agent) rather than reading
// directly. manifestContent is the raw YAML used to derive the node name and
// binary path; it is not written to disk. Both checksums must follow the
// "sha256:<hex>" format produced by checksumFile.
func (r *Registry) AddEntry(manifestPath, manifestContent, manifestChecksum, binaryPath, binaryChecksum string) error {
	if !strings.HasSuffix(manifestPath, ".manifest.spore.yaml") {
		return fmt.Errorf("registry: %q is not a .manifest.spore.yaml file", manifestPath)
	}

	m, err := manifest.ParseManifest(manifestContent)
	if err != nil {
		return fmt.Errorf("registry: invalid manifest %q: %w", manifestPath, err)
	}
	m.Path = manifestPath

	entry := registryEntry{
		Name:           m.Name,
		Manifest:       manifestPath,
		Checksum:       manifestChecksum,
		Binary:         binaryPath,
		BinaryChecksum: binaryChecksum,
	}

	replaced := false
	for i, e := range r.entries {
		if e.Manifest == manifestPath {
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

// ManifestVerified reports whether the manifest at manifestPath was
// successfully verified (both manifest and binary checksums confirmed) during
// Registry.Open(). Returns false for entries that were kept but not readable
// at startup (user-space files); those require connect-time verification.
func (r *Registry) ManifestVerified(manifestPath string) bool {
	for _, e := range r.entries {
		if e.Manifest == manifestPath {
			return e.verified
		}
	}
	return false
}

// ManifestChecksumFor returns the stored manifest checksum for the given
// manifest path. Used to verify the manifest at connect time when it was not
// readable at startup.
func (r *Registry) ManifestChecksumFor(manifestPath string) (checksum string, ok bool) {
	for _, e := range r.entries {
		if e.Manifest == manifestPath {
			return e.Checksum, e.Checksum != ""
		}
	}
	return "", false
}

// BinaryChecksumFor returns the stored binary path and its expected checksum
// for the given manifest path. Returns ok=false when no binary is registered
// (e.g. system nodes with app: n/a).
func (r *Registry) BinaryChecksumFor(manifestPath string) (binaryPath string, checksum string, ok bool) {
	for _, e := range r.entries {
		if e.Manifest == manifestPath {
			if e.Binary == "" || e.BinaryChecksum == "" {
				return "", "", false
			}
			return e.Binary, e.BinaryChecksum, true
		}
	}
	return "", "", false
}


