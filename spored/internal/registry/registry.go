package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"spored/internal/manifest"
	"spored/internal/utilities"
	"strings"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	paths []string
}

func (r *Registry) Open() error {
	
	registryPath, err := RegistryPath()
	if err != nil {
		return err
	}

	EnsureRegistryDir(registryPath)
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		file, err := os.Create(registryPath)
		if err != nil {
			return err
		}
		file.Close()
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		return err
	}

	r.paths = make([]string, 0)
	for _, line := range strings.Split(string(data), "\n") {
		if line != "" {
			r.paths = append(r.paths, line)
		}
	}
	
	return nil
}

func (r *Registry) Paths() []string {
	return r.paths
}

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

	// Resolve app: to an absolute path before writing the stored copy.
	// Relative paths in app: are defined (SPEC §8.2) as relative to the source
	// manifest's directory. After copying, m.Path points to the system copy, so
	// we must canonicalise now while the source directory is still known.
	resolvedApp, err := utilities.ResolveAppPath(m.App, filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("registry: resolve app path: %w", err)
	}
	m.App = resolvedApp

	// Copy the manifest into the daemon-owned nodes directory so _spore can
	// always read it, regardless of where the source file lives.
	nodesDir, err := ManifestsDir()
	if err != nil {
		return fmt.Errorf("registry: nodes dir: %w", err)
	}
	if err := os.MkdirAll(nodesDir, 0755); err != nil {
		return fmt.Errorf("registry: create nodes dir: %w", err)
	}
	systemPath := filepath.Join(nodesDir, m.ID+".manifest.spore.yaml")
	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("registry: serialise manifest: %w", err)
	}
	if err := os.WriteFile(systemPath, data, 0644); err != nil {
		return fmt.Errorf("registry: write manifest copy: %w", err)
	}

	r.paths = append(r.paths, systemPath)
	return r.save()
}

func (r *Registry) Remove(path string) error {
	newPaths := make([]string, 0, len(r.paths))
	for _, p := range r.paths {
		if p != path {
			newPaths = append(newPaths, p)
		}
	}
	r.paths = newPaths
	if err := r.save(); err != nil {
		return err
	}
	// Clean up the managed copy if it lives in the nodes directory.
	if nodesDir, err := ManifestsDir(); err == nil && strings.HasPrefix(path, nodesDir) {
		_ = os.Remove(path)
	}
	return nil
}

func (r *Registry) save() error {
	registryPath, err := RegistryPath()
	if err != nil {
		return err
	}
	content := strings.Join(r.paths, "\n")
	if len(r.paths) > 0 {
		content += "\n"
	}
	return os.WriteFile(registryPath, []byte(content), 0600)
}
