// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package manifest

import (
	"path/filepath"
	reg_el "spored/internal/registry/element"
	"spored/internal/utilities/file"
	"spored/internal/utilities/status"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// manifest context interface
// to avoid package cycling
type inotifier interface {
	Announce(*Manifest)
	Withdraw(*Manifest)
}

//
//
// manifest
// implementation
//

type Manifest struct {
	ID          string          `yaml:"id"`
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Trust       Trust           `yaml:"trust"`
	Schema      string          `yaml:"schema"`
	Version     string          `yaml:"version"`
	App         string          `yaml:"app"`
	Launch      Launch          `yaml:"launch"`
	Namespace   Namespace       `yaml:"namespace"`
	Witness     bool            `yaml:"witness"`
	Roles       []string        `yaml:"roles"`
	Api         []Command       `yaml:"api"`
	Topics      []Topic         `yaml:"topics"`
	Permissions []Permission    `yaml:"permissions"`

	Status           status.Status
	Path             string
	ExpectedChecksum string
	notifier         inotifier
	once             sync.Once
}

type Command struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Risk        Risk      `yaml:"risk"`
	Usage       []string  `yaml:"usage"`
	Inputs      *[]Input  `yaml:"inputs"`
	Outputs     *[]Output `yaml:"outputs"`
	Notes       *[]string `yaml:"notes"`
}

type Topic struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Risk        Risk      `yaml:"risk"`
	Usage       []string  `yaml:"usage"`
	Outputs     *[]Output `yaml:"outputs"`
	Notes       *[]string `yaml:"notes"`
}

type Permission struct {
	Name    string   `yaml:"name"`
	Reasons []string `yaml:"reasons"`
}

type Input struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

type Output struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

func New(registry *reg_el.Element, notifier inotifier) *Manifest {
	m := &Manifest{
		Status:           status.New(),
		Path:             registry.Manifest,
		ExpectedChecksum: registry.Checksum,
		notifier:        notifier,
	}

	if !file.Exists(m.Path) {
		m.Status.Set(status.Missing)
	}

	return m
}

// applyDefaults fills fields left blank in the YAML with their default values.
func (m *Manifest) applyDefaults() {
	if m.Namespace == "" {
		m.Namespace = Hyphae
	}
	if m.Launch == "" {
		m.Launch = Auto
	}
}

func (m *Manifest) Close() {
	if m.notifier != nil {
		m.notifier.Withdraw(m)
	}
}

func (m *Manifest) Verify() {
	if m.Status.Get() == status.Missing {
		return
	}

	if m.Trust == Developer {
		m.Status.Set(status.RequiresDeveloper)
		return
	}

	if !file.IsReadable(m.Path) {
		m.Status.Set(status.RequiresUserSpace)
		return
	}

	checksum, err := file.CalculateChecksum(m.Path)
	if err != nil {
		m.Status.Set(status.FailedChecksum)
		return
	}
	expected := m.ExpectedChecksum
	if !strings.HasPrefix(expected, "sha256:") {
		expected = "sha256:" + expected
	}
	if checksum != expected {
		m.Status.Set(status.FailedChecksum)
		return
	}

	m.Status.Set(status.Verified)
}

func (m *Manifest) Load() {
	contents, err := file.Read(m.Path)
	if err != nil {
		return
	}

	err = yaml.Unmarshal([]byte(contents), m)
	if err != nil {
		m.Status.Set(status.Malformed)
		return
	}
	m.applyDefaults()

	m.once.Do(func() {
		if m.notifier != nil {
			m.notifier.Announce(m)
		}
	})
}

func (m *Manifest) LoadContent(content string) {
	err := yaml.Unmarshal([]byte(content), m) // nolint:errcheck
	if err != nil {
		m.Status.Set(status.Malformed)
		return
	}
	m.applyDefaults()

	m.once.Do(func() {
		if m.notifier != nil {
			m.notifier.Announce(m)
		}
	})
}

//
//
// imanifest
//

func (m *Manifest) GetId() string {
	return m.ID
}

func (m *Manifest) GetName() string {
	return m.Name
}

func (m *Manifest) GetTrust() string {
	return string(m.Trust)
}

func (m *Manifest) GetManifestPath() string {
	return m.Path
}

func (m *Manifest) GetManifestChecksum() string {
	return m.ExpectedChecksum
}

func (m *Manifest) GetBinaryPath() string {
	dir := filepath.Dir(m.Path)
	path := filepath.Join(dir, m.App)
	path = filepath.Clean(path)
	return file.ExecutablePath(path)
}
