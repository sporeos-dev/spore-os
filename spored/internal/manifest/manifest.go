package manifest

import (
	"path/filepath"
	"spored/internal/registry"
	"spored/internal/utilities/file"
	"spored/internal/utilities/status"
	"strings"

	"gopkg.in/yaml.v3"
)

type Manifest struct {
	ID          string          `yaml:"id"`
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Trust		Trust			`yaml:"trust"`
	Schema      string          `yaml:"schema"`
	Version     string          `yaml:"version"`
	App         string          `yaml:"app"`
	Autostart   bool            `yaml:"autostart"`
	Witness		bool			`yaml:"witness"`
	Api         []Command       `yaml:"api"`
	Topics		[]Topic			`yaml:"topics"`	
	Errors      []ManifestError `yaml:"errors"`

	Status status.Status
	Path string
	ExpectedChecksum string
}

type Command struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Risk		Risk	  `yaml:"risk"`
	Usage       []string  `yaml:"usage"`
	Inputs      *[]Input  `yaml:"inputs"`
	Outputs     *[]Output `yaml:"outputs"`
	Notes       *[]string `yaml:"notes"`
}

type Topic struct {
	Name		string	  `yaml:"name"`
	Description string	  `yaml:"description"`
	Risk		Risk	  `yaml:"risk"`
	Usage		[]string  `yaml:"usage"`
	Outputs		*[]Output `yaml:"outputs"`
	Notes		*[]string `yaml:"notes"`
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

type ManifestError struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Examples    []string `yaml:"examples,omitempty"`
}

func New(registry *registry.Element) *Manifest {
	m := &Manifest{
		Status: status.New(), 
		Path: registry.Manifest,
		ExpectedChecksum: registry.Checksum,
	}

	if file.Exists(m.Path) == false {
		m.Status.Set(status.Missing)
	}

	return m
}

func ManifestFromPath(path string) *Manifest {
	m := &Manifest{
		Status: status.New(),
		Path: path,
	}

	contents, err := file.Read(path)
	if err != nil {
		return nil
	}

	err = yaml.Unmarshal([]byte(contents), m)
	if err != nil {
		return nil
	}

	checksum, sp_err := file.CalculateChecksum(path)
	if sp_err != nil {
		return nil
	}

	m.ExpectedChecksum = checksum
	return m
}

func (m *Manifest) Close() {}

func (m *Manifest) Verify() {
	if file.IsReadable(m.Path) == false {
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
		return
	}
}

func (m *Manifest) LoadContent(content string) {
	yaml.Unmarshal([]byte(content), m)
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
	return path
}
