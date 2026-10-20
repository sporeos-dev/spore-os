// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package permissions

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/pal"
	"spored/internal/utilities/await"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	protectedCaller                  = "hyphae-in-spore"
	hyphaeConfirmationCapability     = "dev.sporeos.HYPHAE.confirm"
	permissionCaller                 = "permission-in-spore"
	permissionConfirmationCapability = "dev.sporeos.shell.confirm"
	permissionTokenLifetime          = 30 * time.Second
)

type permit struct {
	origin     string
	capability string
	expiresAt  time.Time
}

type Manager struct {
	index   atomic.Int64
	pending *await.Pending

	mu                sync.RWMutex
	allowances        map[string][]string
	permits           map[string]permit
	sysNodes          []string // granted outright
	benignCommands    []string // granted outright
	benignTopics      []string // granted outright
	adminCommands     []string
	protectedCommands []string

	bus    ibus
	hyphae ihyphae
	nodes  inodes
	spore  ispore
}

func New() *Manager {
	m := &Manager{
		pending:    await.New(error.Permission),
		allowances: make(map[string][]string),
		permits:    make(map[string]permit),
	}

	if err := m.Load(); err != nil {
		// a missing/corrupt file just means no allowances are known yet;
		// deny-by-default is safe, so we log and carry on rather than exit.
		slog.Warn("Failed to load permissions, starting with an empty allowance set", "error", err)
	}

	return m
}

func (m *Manager) Close() {}

func (m *Manager) Set(bus ibus, hyphae ihyphae, nodes inodes, spore ispore) {
	m.bus = bus
	m.hyphae = hyphae
	m.nodes = nodes
	m.spore = spore

	m.bus.Register(m)
}

func (m *Manager) List(nodeid string) ([]string, *error.Error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allowances, ok := m.allowances[nodeid]
	if !ok {
		return nil, error.New(
			error.Missing,
			error.Permission,
			"node not found",
			out.Pair("node", nodeid))
	}

	return allowances, nil
}

func (m *Manager) RegisterNode(man *manifest.Manifest) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if man == nil {
		return
	}

	switch man.Trust {
	case manifest.System:
		m.sysNodes = append(m.sysNodes, man.ID)
	default:
		// do nothing for developer, trusted, standard, or untrusted nodes
	}

	for _, command := range man.Api {
		capability := man.ID + "." + command.Name
		if command.Risk == manifest.Benign {
			m.benignCommands = append(m.benignCommands, capability)
		}
		if command.Risk == manifest.Admin {
			m.adminCommands = append(m.adminCommands, capability)
		}
		if command.Risk == manifest.Protected {
			m.protectedCommands = append(m.protectedCommands, capability)
		}
	}

	for _, topic := range man.Topics {
		if topic.Risk == manifest.Benign {
			m.benignTopics = append(m.benignTopics, topic.Name)
		}
	}
}

func (m *Manager) Grant(nodeid string, command string) *error.Error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if slices.Contains(m.allowances[nodeid], command) {
		return nil
	}
	m.allowances[nodeid] = append(m.allowances[nodeid], command)

	return m.save()
}

func (m *Manager) Revoke(nodeid string, command string) *error.Error {
	m.mu.Lock()
	defer m.mu.Unlock()
	commands := m.allowances[nodeid]
	for i, c := range commands {
		if c == command {
			m.allowances[nodeid] = append(commands[:i], commands[i+1:]...)
			break
		}
	}

	return m.save()
}

// RemoveNode removes every persisted capability grant for an uninstalled node.
func (m *Manager) RemoveNode(nodeid string) *error.Error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.allowances, nodeid)
	return m.save()
}

func (m *Manager) Request(nodeid string, capability string, reason []string) (bool, *error.Error) {
	granted, err := m.confirm(nodeid, capability, reason)
	if err != nil || !granted {
		return granted, err
	}
	if err := m.Grant(nodeid, capability); err != nil {
		return false, err
	}
	return true, nil
}

func (m *Manager) RequestPermit(nodeid string, capability string, reason []string) (string, bool, *error.Error) {
	m.mu.RLock()
	isAdmin := slices.Contains(m.adminCommands, capability)
	isProtected := slices.Contains(m.protectedCommands, capability)
	m.mu.RUnlock()
	if isAdmin || isProtected {
		return "", false, error.New(
			error.NotPermitted,
			error.Permission,
			"permission permits cannot authorize this capability",
			out.Pair("capability", capability))
	}

	granted, err := m.confirm(nodeid, capability, reason)
	if err != nil || !granted {
		return "", granted, err
	}

	token, err := newPermitToken()
	if err != nil {
		return "", false, err
	}
	m.mu.Lock()
	if m.permits == nil {
		m.permits = make(map[string]permit)
	}
	for existingToken, existingPermit := range m.permits {
		if !time.Now().Before(existingPermit.expiresAt) {
			delete(m.permits, existingToken)
		}
	}
	m.permits[token] = permit{
		origin:     nodeid,
		capability: capability,
		expiresAt:  time.Now().Add(permissionTokenLifetime),
	}
	m.mu.Unlock()
	return token, true, nil
}

func (m *Manager) confirm(nodeid string, capability string, reason []string) (bool, *error.Error) {
	handle := m.handle()

	// e.g.
	// dev.sporeos.shell.confirm body="Grant node [nodeid] permission to [capability]?" lines=[ "Reasons", " - reason A", " - reason B"  ]
	var conf strings.Builder
	conf.WriteString("dev.sporeos.shell.confirm body=\"Grant node [")
	conf.WriteString(nodeid)
	conf.WriteString("] permission to [")
	conf.WriteString(capability)
	conf.WriteString("]?\"")
	if len(reason) > 0 {
		conf.WriteString(" lines=[ \"Reasons\"")
		for _, reason := range reason {
			conf.WriteString(", \" - ")
			conf.WriteString(reason)
			conf.WriteString("\"")
		}
		conf.WriteString(" ]")
	}
	conf.WriteString(" ~")
	conf.WriteString(handle)

	raw := conf.String()
	msg, ok := message.Request(raw, m.Id())
	if !ok {
		return false, error.New(
			error.Malformed,
			error.Permission,
			"failure to request permission")
	}
	ch := m.pending.Await(handle)
	err := m.bus.Request(msg)
	if err != nil {
		m.pending.Delete(handle)
		return false, err
	}
	response, err := m.pending.WaitFor(handle, ch)
	if err != nil {
		return false, err
	}
	if response.Flag("error") {
		return false, error.New(error.Generic, error.Permission, response.ArgIf("what", "permission request failed"))
	}
	if !response.Flag("yes") {
		return false, nil
	}
	return true, nil
}

func (m *Manager) Can(msg iface.Message, capability string) (bool, *error.Error) {
	actor := msg.Actor()
	origin := msg.Origin()

	m.mu.RLock()
	isProtected := slices.Contains(m.protectedCommands, capability)
	isAdmin := slices.Contains(m.adminCommands, capability)
	allowed := slices.Contains(m.sysNodes, origin) ||
		slices.Contains(m.benignCommands, capability) ||
		slices.Contains(m.benignTopics, capability) ||
		allows(m.allowances[origin], capability)
	m.mu.RUnlock()

	if isProtected {
		return actor == protectedCaller, nil
	}
	if origin == permissionCaller && capability == permissionConfirmationCapability {
		return true, nil
	}
	if origin == protectedCaller && capability == hyphaeConfirmationCapability {
		return true, nil
	}
	if allowed {
		if !isAdmin {
			return true, nil
		}
		if m.hyphae == nil {
			return false, error.New(
				error.Generic,
				error.Permission,
				"Hyphae confirmation is unavailable",
				out.Pair("capability", capability))
		}
		return m.hyphae.Confirm(origin, capability)
	}
	if isAdmin {
		return false, nil
	}
	token, ok := msg.Arg("permission.token")
	if !ok {
		return false, nil
	}
	return m.consumePermit(origin, capability, token), nil
}

func (m *Manager) consumePermit(origin string, capability string, token string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	permit, ok := m.permits[token]
	if !ok || permit.origin != origin || permit.capability != capability || !time.Now().Before(permit.expiresAt) {
		return false
	}
	delete(m.permits, token)
	return true
}

func allows(allowances []string, capability string) bool {
	for _, allowance := range allowances {
		if allowance == capability || strings.HasSuffix(capability, "."+allowance) {
			return true
		}
	}
	return false
}

// Load reads allowances from disk, replacing the in-memory set.
func (m *Manager) Load() *error.Error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.load()
}

func (m *Manager) load() *error.Error {
	data, err := os.ReadFile(pal.FilePermissions())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return error.New(
			error.Missing,
			error.Permission,
			"unable to read permissions",
			out.Pair("err", err.Error()))
	}

	allowances := make(map[string][]string)
	err = yaml.Unmarshal(data, &allowances)
	if err != nil {
		return error.New(
			error.Malformed,
			error.Permission,
			"unable to parse permissions",
			out.Pair("err", err.Error()))
	}

	m.allowances = allowances
	return nil
}

// Save writes the current allowances to disk.
func (m *Manager) Save() *error.Error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.save()
}

func (m *Manager) save() *error.Error {
	data, err := yaml.Marshal(m.allowances)
	if err != nil {
		return error.New(
			error.Malformed,
			error.Permission,
			"unable to serialize permissions",
			out.Pair("err", err.Error()))
	}

	if err := os.MkdirAll(filepath.Dir(pal.FilePermissions()), 0700); err != nil {
		return error.New(
			error.Generic,
			error.Permission,
			"unable to create permissions directory",
			out.Pair("err", err.Error()))
	}

	err = os.WriteFile(pal.FilePermissions(), data, 0600) // only readable/writable by the spore
	if err != nil {
		return error.New(
			error.Generic,
			error.Permission,
			"unable to write permissions",
			out.Pair("err", err.Error()))
	}

	return nil
}

//
//
// private
// permission call
// helper
//

func (h *Manager) handle() string {
	return fmt.Sprintf("permission-%d", h.index.Add(1))
}

//
//
// INode
//

func (m *Manager) Id() string {
	return permissionCaller
}

func newPermitToken() (string, *error.Error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", error.New(
			error.Generic,
			error.Permission,
			"unable to create permission permit",
			out.Pair("err", err.Error()))
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func (m *Manager) IsConnected() bool {
	return true
}

func (m *Manager) IsWitness() bool {
	return false
}

func (m *Manager) GetManifest() *manifest.Manifest {
	return nil
}

func (m *Manager) Receive(msg iface.Message) *error.Error {
	return m.pending.Receive(msg)
}

func (m *Manager) Witness(msg iface.Message) {}
