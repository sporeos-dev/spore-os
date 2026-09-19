package permissions

import (
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

	"gopkg.in/yaml.v3"
)

type Manager struct {
	index   atomic.Int64
	pending *await.Pending

	mu sync.RWMutex
	allowances map[string][]string
	devNodes []string // granted outright
	sysNodes []string // granted outright
	benignCommands []string // granted outright
	benignTopics []string // granted outright

	bus ibus
	hyphae ihyphae
	nodes inodes
	spore ispore
}

func New() *Manager {
	m := &Manager{
		pending: await.New(error.Permission),
		allowances: make(map[string][]string),
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
	case manifest.Developer:
		m.devNodes = append(m.devNodes, man.ID)
	case manifest.System:
		m.sysNodes = append(m.sysNodes, man.ID)
	default:
		// do nothing for trusted, standard, untrusted nodes
	}

	for _, command := range man.Api {
		if command.Risk == manifest.Benign {
			m.benignCommands = append(m.benignCommands, command.Name)
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

func (m *Manager) Request(nodeid string, capability string, reason []string, ) (bool, *error.Error) {
	handle := m.handle()

	// e.g.
	// dev.sporeos.shell.confirm body="Grant node [nodeid] permission to [capability]?" lines=[ "Reasons", " - reason A", " - reason B"  ]
	var conf strings.Builder; 
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
	return response.Flag("yes"), nil
}

func (m *Manager) Can(nodeid string, capability string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Contains(m.sysNodes, nodeid) ||
		   slices.Contains(m.devNodes, nodeid) || 
		   slices.Contains(m.benignCommands, capability) ||
		   slices.Contains(m.benignTopics, capability) ||
		   slices.Contains(m.allowances[nodeid], capability)
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
	return "permission-in-spore"
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
