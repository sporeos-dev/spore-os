package nodes

import (
	"spored/internal/utilities/file"

	"gopkg.in/yaml.v3"
)

type manifest struct {
	ID          string          `yaml:"id"`
	Name        string          `yaml:"name"`
	Description string          `yaml:"description"`
	Trust		trust			`yaml:"trust"`
	Schema      string          `yaml:"schema"`
	Version     string          `yaml:"version"`
	App         string          `yaml:"app"`
	Autostart   bool            `yaml:"autostart"`
	Witness		bool			`yaml:"witness"`
	Api         []command       `yaml:"api"`
	Topics		[]topic			`yaml:"topics"`	
	Errors      []manifestError `yaml:"errors"`

	status status
	path string
	expectedChecksum string
}

type command struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Risk		risk	  `yaml:"risk"`
	Usage       []string  `yaml:"usage"`
	Inputs      *[]input  `yaml:"inputs"`
	Outputs     *[]output `yaml:"outputs"`
	Notes       *[]string `yaml:"notes"`
}

type topic struct {
	Name		string	  `yaml:"name"`
	Description string	  `yaml:"description"`
	Risk		risk	  `yaml:"risk"`
	Usage		[]string  `yaml:"usage"`
	Outputs		*[]output `yaml:"outputs"`
	Notes		*[]string `yaml:"notes"`
}

type input struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

type output struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

type manifestError struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Examples    []string `yaml:"examples,omitempty"`
}

func newManifest(registry *registryElement) *manifest {
	m := &manifest{
		status: newStatus(), 
		path: registry.Manifest,
		expectedChecksum: registry.Checksum,
	}

	if file.Exists(m.path) == false {
		m.status.set(Missing)
	}

	return m
}

func (m *manifest) close() {}

func (m *manifest) verify() {
	if file.IsReadable(m.path) == false {
		m.status.set(RequiresUserSpace)
		return
	}

	checksum, err := file.CalculateChecksum(m.path)
	if err != nil {
		m.status.set(FailedChecksum)
		return
	}
	if checksum != m.expectedChecksum {
		m.status.set(FailedChecksum)
		return
	}

	m.status.set(Verified)
}

func (m *manifest) load() {
	contents, err := file.Read(m.path)
	if err != nil {
		return
	}

	err = yaml.Unmarshal([]byte(contents), m)
	if err != nil {
		return
	}
}
