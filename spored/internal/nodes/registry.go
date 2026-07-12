package nodes

import (
	"log/slog"
	"os"
	"spored/internal/pal"
	"sync"

	"gopkg.in/yaml.v3"
)

type registry struct {
	mu       sync.Mutex
	Version  int                `yaml:"version"`
	Elements []*registryElement `yaml:"nodes"`
}

type registryElement struct {
	Name           string `yaml:"name"`
	Manifest       string `yaml:"manifest"`
	Checksum       string `yaml:"checksum"`
	Binary         string `yaml:"binary"`
	BinaryChecksum string `yaml:"binary_checksum"`
}

func newRegistry() *registry {
	
	reg := &registry{}
	
	err := reg.load()
	if err != nil {
		slog.Error("Failed to load registry", "error", err)
		os.Exit(1)
	}

	return reg
}

func (r *registry) close() {}

func (r *registry) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := os.ReadFile(pal.FileRegistry())
	if err != nil {
		slog.Error("Failed to load registry", "error", err)
		return err
	}

	err = yaml.Unmarshal(data, r)
	if err != nil {
		slog.Error("Failed to parse registry", "error", err)
		return err
	}
	
	return nil
}

func (r *registry) save() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := yaml.Marshal(r)
	if err != nil {
		slog.Error("Failed to marshal registry", "error", err)
		return err
	}

	err = os.WriteFile(pal.FileRegistry(), data, 0600) // only readable/writable by the spore
	if err != nil {
		slog.Error("Failed to save registry", "error", err)
		return err
	}
	return nil
}