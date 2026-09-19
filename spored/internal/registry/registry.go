package registry

import (
	"log/slog"
	"os"
	"sync"

	"spored/internal/pal"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"

	"gopkg.in/yaml.v3"
)

type Registry struct {
	mu       sync.RWMutex
	Version  int        `yaml:"version"`
	Elements []*Element `yaml:"nodes"`
}

func New() *Registry {
	
	reg := &Registry{}
	
	err := reg.Load()
	if err != nil {
		slog.Error("Failed to load registry", "error", err)
		os.Exit(1)
	}

	return reg
}

func (r *Registry) Close() {}

func (r *Registry) Add(el *Element) *error.Error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Elements = append(r.Elements, el)
	return r.save()
}

func (r *Registry) Remove(id string) *error.Error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, el := range r.Elements {
		if id != el.ID {
			continue
		}
		r.Elements = append(r.Elements[:i], r.Elements[i+1:]...)
		return r.save()
	}

	return error.New(
		error.Missing, 
		error.Registry, 
		"unable to remove element", 
		out.Pair("node", id))
}

func (r *Registry) Load() *error.Error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.load()
}

func (r *Registry) load() *error.Error {
	
	data, err := os.ReadFile(pal.FileRegistry())
	if err != nil {
		return error.New(
			error.Missing,
			error.Registry,
			"unable to read registry",
			out.Pair("err", err.Error()))
	}

	err = yaml.Unmarshal(data, r)
	if err != nil {
		return error.New(
			error.Malformed,
			error.Registry,
			"unable to parse registry",
			out.Pair("err", err.Error()))
	}
	
	return nil
}

func (r *Registry) Save() *error.Error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.save()
}

func (r *Registry) save() *error.Error {
	data, err := yaml.Marshal(r)
	if err != nil {
		return error.New(
			error.Malformed,
			error.Registry,
			"unable to serialize registry",
			out.Pair("err", err.Error()))
	}

	err = os.WriteFile(pal.FileRegistry(), data, 0600) // only readable/writable by the spore
	if err != nil {
		return error.New(
			error.Generic,
			error.Registry, 
			"unable to write registry",
			out.Pair("err", err.Error()))
	}
	
	return nil
}