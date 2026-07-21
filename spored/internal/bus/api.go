package bus

import (
	"spored/internal/message"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"sync"
)

type api struct {
	mu sync.RWMutex
	commands map[string]string
	nodes map[string]INode
}

func newApi() *api {
	return &api {
		commands: make(map[string]string),
		nodes: make(map[string]INode),
	}
}

func (a *api) close() {}

func (a *api) register(n INode) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.nodes[n.Id()] = n
	manifest := n.GetManifest()
	if manifest != nil {
		for _, command := range manifest.Api {
			a.commands[command.Name] = n.Id()
		}
	}
}

func (a *api) unregister(n INode) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.nodes, n.Id())
	manifest := n.GetManifest()
	if manifest != nil {
		for _, command := range manifest.Api {
			delete(a.commands, command.Name)
		}
	}
}

func (a *api) request(msg message.Message) *error.Error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if nodeid, ok := a.commands[msg.Command()]; ok {
		if node, ok := a.nodes[nodeid]; ok {
			return node.Receive(msg)
		}
	}

	return error.New(
		error.Missing,
		error.Message,
		"command not found",
		out.Pair("command", msg.Command()))
}
