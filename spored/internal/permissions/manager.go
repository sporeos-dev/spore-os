package permissions

import (
	"fmt"
	"slices"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/utilities/await"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"strings"
	"sync"
)

type Manager struct {
	index   int
	pending *await.Pending

	mu sync.RWMutex
	allowances map[string][]string

	bus ibus
	hyphae ihyphae
	nodes inodes
	spore ispore
}

func New() *Manager {
	return &Manager{
		index: 0,
		pending: await.New(error.Permission),
		allowances: make(map[string][]string),
	}
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

func (m *Manager) Grant(nodeid string, command string) *error.Error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.allowances[nodeid] = append(m.allowances[nodeid], command)

	return nil
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

	return nil
}

func (m *Manager) Request(nodeid string, capability string, reason []string, ) (Value, *error.Error) {
	handle := m.handle()

	title := "Permission Request"
	var description strings.Builder; 
	description.WriteString(fmt.Sprintf("Do you want to grant node [%s] permission to [%s]?", nodeid, capability))
	description.WriteString(`\n\nNote: only grant permissions you believe will be respected; there is no current enforcement policy.`)
	description.WriteString(`\n\nStated Reasons`)
	for _, el := range reason {
		description .WriteString(`\n - ` + el)
	}
	raw := fmt.Sprintf(`dialog.alert title="%s" description="%s" entries=[Always Once No] ~%s`, title, description.String(), handle)
	
	msg, ok := message.Request(raw, m.Id())
	if !ok {
		return No, error.New(
			error.Malformed,
			error.Permission,
			"failure to request permission")
	}
	ch := m.pending.Await(handle)
	err := m.bus.Request(msg)
	if err != nil {
		m.pending.Delete(handle)
		return No, err
	}
	response, err := m.pending.WaitFor(handle, ch)
	if err != nil {
		return No, err
	}
	if response.Flag("error") {
		return No, error.New(error.Generic, error.Permission, response.ArgIf("what", "permission request failed"))
	}
	entry, ok := response.Arg("entry")
	if ok {
		return No, error.New(
			error.Missing,
			error.Permission,
			"missing in response",
			out.Pair("argument", "entry"))
	}
	return Value(entry), nil
}

func (m *Manager) Can(nodeid string, capability string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return slices.Contains(m.allowances[nodeid], capability)
}

//
//
// private
// permission call
// helper
//

func (h *Manager) handle() string {
	h.index++
	return fmt.Sprintf("permission-%d", h.index)
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
