package bus

import (
	"spored/internal/iface"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"strings"
	"sync"
)

type api struct {
	mu sync.RWMutex
	nodes map[string]INode
	commands map[string]string // fqname -> node id
	abbrevs *abbreviationIndex
}

func newApi() *api {
	return &api {
		nodes: make(map[string]INode),
		commands: make(map[string]string),
		abbrevs: newAbbreviationIndex(),
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
			fqname := n.Id() + "." + command.Name
			a.commands[fqname] = n.Id()
			a.abbrevs.add(fqname)
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
			fqname := n.Id() + "." + command.Name
			delete(a.commands, fqname)
			a.abbrevs.remove(fqname)
		}
	}
}

// fullyQualifiedRequest resolves a (possibly abbreviated) command to its
// canonical fqname, returning it unchanged if it can't be resolved.
func (a *api) fullyQualifiedRequest(command string) string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if fqname, _, ok := a.abbrevs.resolve(command); ok {
		return fqname
	}
	return command
}

func (a *api) request(msg iface.Message) *error.Error {
	a.mu.RLock()

	key := msg.Capability()

	fqname, candidates, ok := a.abbrevs.resolve(key)
	if !ok {
		a.mu.RUnlock()
		if len(candidates) > 0 {
			return error.New(
				error.Collision,
				error.Bus,
				"ambiguous command, use a more qualified name",
				out.Pair("command", key),
				out.Pair("candidates", strings.Join(candidates, ", "))).
				WithMessage(msg)
		}
		return error.New(
			error.Missing,
			error.Bus,
			"command not found",
			out.Pair("command", key)).
			WithMessage(msg)
	}

	nodeid, ok := a.commands[fqname]
	if !ok {
		a.mu.RUnlock()
		return error.New(
			error.Missing,
			error.Bus,
			"command not found",
			out.Pair("command", key)).
			WithMessage(msg)
	}

	node, ok := a.nodes[nodeid]
	if !ok {
		a.mu.RUnlock()
		return error.New(
			error.Missing,
			error.Bus,
			"node not found",
			out.Pair("command", key)).
			WithMessage(msg)
	}

	a.mu.RUnlock()
	return node.Receive(msg)
}
