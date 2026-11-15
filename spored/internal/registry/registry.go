// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package registry

import (
	"os"
	"sync"

	"spored/internal/base"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	registry_event "spored/internal/event/registry"
	"spored/internal/pal"
	reg_el "spored/internal/registry/element"
	"spored/internal/utilities/error"
	"spored/internal/utilities/file"
	"spored/internal/utilities/out"
	"spored/internal/witness"

	"gopkg.in/yaml.v3"
)

var _ message_event.IRequestHandler = (*Registry)(nil)

type Registry struct {
	*base.Base
	mu       sync.RWMutex
	Version  int        `yaml:"version"`
	Elements []*reg_el.Element `yaml:"nodes"`
}

func New() *Registry {
	
	reg := &Registry{
		Base: base.New().SetName("registry"),
	}
	
	return reg
}

func (r *Registry) Open() {
	event.Subscribe(r)
}

func (r *Registry) Close() {
	event.Unsubscribe(r)
}

func (r *Registry) Add(el *reg_el.Element) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Elements = append(r.Elements, el)
	saved := r.save()
	if saved {
		ctx := registry_event.New(el)
		event.Dispatch().RegistryAnnouncement.Notify(ctx)
	}
	return saved
}

func (r *Registry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, el := range r.Elements {
		if id != el.ID {
			continue
		}

		r.Elements = append(r.Elements[:i], r.Elements[i+1:]...)
		saved :=r.save()
		if saved {
			ctx := registry_event.New(el)
			event.Dispatch().RegistryWithdrawal.Notify(ctx)
		}
		return saved
	}

	witness.Error(
		error.Missing,
		error.Registry,
		"element not found",
		out.Pair("node", id))

		return false
}

func (r *Registry) Load() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.load()
}

func (r *Registry) load() bool {
	
	data, err := os.ReadFile(pal.FileRegistry())
	if err != nil {
		witness.Error(
			error.Missing,
			error.Registry,
			"unable to read registry",
			out.Pair("err", err.Error()))
		return false
	}

	err = yaml.Unmarshal(data, r)
	if err != nil {
		witness.Error(
			error.Malformed,
			error.Registry,
			"unable to parse registry",
			out.Pair("err", err.Error()))
		return false
	}
	
	for _, el := range r.Elements {
		if el == nil {
			continue
		}
			
		el.Binary = file.ExecutablePath(el.Binary)
		ctx := registry_event.New(el)
		event.Dispatch().RegistryAnnouncement.Notify(ctx)
	}

	return true
}

func (r *Registry) Save() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.save()
}

func (r *Registry) save() bool {
	data, err := yaml.Marshal(r)
	if err != nil {
		witness.Error(
			error.Malformed,
			error.Registry,
			"unable to serialize registry",
			out.Pair("err", err.Error()))
		return false
	}

	err = os.WriteFile(pal.FileRegistry(), data, 0600) // only readable/writable by the spore
	if err != nil {
		witness.Error(
			error.Generic,
			error.Registry, 
			"unable to write registry",
			out.Pair("err", err.Error()))
		return false
	}

	return true
}
