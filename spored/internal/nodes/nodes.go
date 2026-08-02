package nodes

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/permissions"
	"spored/internal/registry"
	"spored/internal/utilities/await"
	"spored/internal/utilities/file"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"strings"
	"sync"
	"time"

	"spored/internal/utilities/error"
)

const handshakeTimeout = 5 * time.Second

type Nodes struct {
	index int
	pending *await.Pending

	registry *registry.Registry

	mu sync.RWMutex
	nodes map[string]*node

	bus ibus
	hyphae ihyphae
	permissions ipermissions
	spore ispore
}

func New() *Nodes {
	nodes := &Nodes{
		index : 0,
		pending: await.New(error.Node),
		registry: registry.New(),
		nodes:   make(map[string]*node),
	}

	nodes.mu.Lock()
	defer nodes.mu.Unlock()

	for _, el := range nodes.registry.Elements {
		manifest := manifest.New(el)
		n := newNode(el, manifest)
		nodes.nodes[n.registry.ID] = n
	}

	return nodes
}

func (n *Nodes) Set(bus ibus, hyphae ihyphae, permissions ipermissions, spore ispore) {
	n.bus = bus
	n.hyphae = hyphae
	n.permissions = permissions
	n.spore = spore

	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, el := range n.nodes {
		el.set(bus, n, permissions)
	}
}

func (n *Nodes) Close() {
	n.registry.Close()
}

func (n *Nodes) Autostart() {
	
	n.mu.RLock()
	defer n.mu.RUnlock()
	for _, el := range n.nodes {
		if el.manifest.Launch == manifest.Auto {
			if el.manifest.Namespace == manifest.Hyphae {
				n.bus.WitnessSpore(fmt.Sprintf("%s skipped autostarting because it requires hyphae spsace (%s)", el.registry.Name, el.registry.ID))
				continue
			}

			err := n.Spawn(el.registry.ID)
			if err != nil {
				el.Error(
					error.New(
						error.Generic,
						error.Node,
						"autostart failure",
						out.Pair("node", el.registry.ID)),
					"",
					"")
				continue
			}

			n.bus.WitnessSpore(fmt.Sprintf("%s autostarted (%s)", el.registry.Name, el.registry.ID))
		}
	}
}

func (n *Nodes) HandleConnection(conn net.Conn) {

	conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetReadDeadline(time.Time{})

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	nodeid, err := reader.ReadString('\n')
	if err != nil {
		sperr := error.New(
			error.HandshakeDenial,
			error.Node,
			"failed initial handshake read",
			out.Pair("error", err.Error()))
		
		n.bus.WitnessSpore(fmt.Sprintf("Unable to connect to node; failed initial read (error: %s)", sperr.Wire()))
		writer.WriteString(sperr.Wire())
		writer.Flush()
		conn.Close()
		return
	}

	nodeid = strings.TrimSpace(nodeid)
	n.bus.WitnessSpore(fmt.Sprintf("Node connecting: %s", nodeid))

	n.mu.RLock()
	defer n.mu.RUnlock()
	node, ok := n.nodes[nodeid]
	if !ok {
		sperr := error.New(
			error.HandshakeDenial,
			error.Node,
			"node not installed",
			out.Pair("node", nodeid))
		
		n.bus.WitnessOut(fmt.Sprintf("Unable to connect to node; node not found (node: %s, error: %s)", nodeid, sperr.Wire()), nodeid)
		writer.WriteString(sperr.Wire())
		writer.Flush()
		conn.Close()
		return
	}

	// Handle the connection with the node
	sp_err := node.handleConnection(conn, reader, writer, n.hyphae)
	if sp_err != nil {
		n.bus.WitnessOut(fmt.Sprintf("Unable to connect to node: failed to handle connection (node: %s, error: %s)", nodeid, sp_err.Wire()), nodeid)
		writer.WriteString(sp_err.Wire())
		writer.Flush()
		conn.Close()
		return;
	}
}

//
//
// external
// interfaces
//

func (n *Nodes) GetNodes() []string {

	n.mu.RLock()
	defer n.mu.RUnlock()

	nodeids := make([]string, 0, len(n.nodes))
	for nodeid := range n.nodes {
		nodeids = append(nodeids, nodeid)
	}

	return nodeids
}

func (n *Nodes) GetManifest(nodeid string) *manifest.Manifest {
	nodeid = n.resolveId(nodeid)

	n.mu.RLock()
	node, ok := n.nodes[nodeid]
	n.mu.RUnlock()
	if !ok { return nil }

	if node.manifest.ID == "" && n.hyphae != nil {
		content, err := n.hyphae.ManifestRead(node.manifest.Path)
		if err == nil {
			node.manifest.LoadContent(content)
			checksum, err := n.hyphae.HashFile(node.manifest.Path)
			if err == nil {
				if node.manifest.ExpectedChecksum == checksum {
					node.manifest.Status.Set(status.Verified)
				} else {
					node.manifest.Status.Set(status.FailedChecksum)
				}
			}
			n.bus.Register(node)
		}
	}

	return node.manifest
}

func (n *Nodes) Install(path string) *error.Error {

	if !file.Exists(path) {
		return error.New(
			error.Missing, 
			error.Node,
			"unable to install due to bad path",
			out.Pair("path", path))
	}

	var node *node = nil

	// readable, handle via spore
	if file.IsReadable(path) {
		manifest := manifest.ManifestFromPath(path)
		if manifest == nil {
			return error.New(
				error.Generic,
				error.Node,
				"read failure",
				out.Pair("path", path))
		}

		if msg := manifest.ValidateReservedLanguage(); msg != "" {
			return error.New(
				error.ReservedLanguage,
				error.Node,
				msg,
				out.Pair("node", manifest.ID))
		}

		if !n.acceptInstallationWarning(manifest) {
			return error.New(
				error.UserDenial,
				error.Node,
				"installation cancelled due to trust misalignment",
				out.Pair("node", manifest.ID),
				out.Pair("trust", string(manifest.Trust)))
		}

		registryElement := registry.ElementFromManifest(manifest)
		if registryElement == nil {
			return error.New(
				error.Generic,
				error.Node,
				"failed to create registry element",
				out.Pair("path", path))
		}

		err := n.registry.Add(registryElement)
		if err != nil {
			return err
		}

		node = newNode(registryElement, manifest)
		node.set(n.bus, n, n.permissions)
		n.nodes[node.registry.ID] = node

	// not readable, handle via hyphae
	} else {

		manifest, registry, err := n.hyphae.PrepareForInstallation(path)
		if err != nil {
			return err
		}

		if msg := manifest.ValidateReservedLanguage(); msg != "" {
			return error.New(
				error.ReservedLanguage,
				error.Node,
				msg,
				out.Pair("node", manifest.ID))
		}

		if !n.acceptInstallationWarning(manifest) {
			return error.New(
				error.UserDenial,
				error.Node,
				"installation cancelled due to trust misalignment",
				out.Pair("node", manifest.ID),
				out.Pair("trust", string(manifest.Trust)))
		}

		err = n.registry.Add(registry)
		if err != nil {
			return err
		}

		node = newNode(registry, manifest)
		node.set(n.bus, n, n.permissions)
		n.nodes[node.registry.ID] = node
	}

	if node != nil && 
		node.manifest.Trust != manifest.System &&
		node.manifest.Trust != manifest.Developer &&
		node.manifest.Trust != manifest.Trusted {
		for _, el := range node.manifest.Permissions {
			perm, err := n.permissions.Request(node.registry.ID, el.Name, el.Reasons)
			if err != nil {
				n.bus.WitnessSpore(fmt.Sprintf("Permission failure during install (%s)", err.Error()))
				continue
			}
			switch perm {
			case permissions.Always, permissions.Once:
				n.bus.WitnessSpore(fmt.Sprintf("Permission %s granted to %s.", el.Name, node.registry.ID))	
			case permissions.No, permissions.Never:
				n.bus.WitnessSpore(fmt.Sprintf("Permssion %s denied for %s.", el.Name, node.registry.ID))	
			}
		}
	}

	return nil
}

func (n *Nodes) Uninstall(nodeid string) *error.Error {
	nodeid = n.resolveId(nodeid)
	
	n.mu.Lock()
	defer n.mu.Unlock()

	node, ok := n.nodes[nodeid]
	if !ok {
		return error.New(
			error.Missing,
			error.Node,
			"cannot uninstall; node not installed",
			out.Pair("node", nodeid))
	}

	delete(n.nodes, nodeid)
	return n.registry.Remove(node.registry.ID)
}

func (n *Nodes) Spawn(nodeid string) *error.Error {

	nodeid = n.resolveId(nodeid)
	n.bus.WitnessSpore(fmt.Sprintf("Spawning %s", nodeid))

	n.mu.RLock()
	defer n.mu.RUnlock()

	node, ok := n.nodes[nodeid]
	if !ok {
		return error.New(
			error.Missing,
			error.Node,
			"cannot spawn; node not installed")
	}

	if file.IsReadable(node.registry.Binary) {
		if node.manifest.Namespace == manifest.Hyphae {
			err := n.hyphae.Spawn(node.registry.Binary)
			if err != nil {
				return error.New(
					error.Generic,
					error.Node,
					"failed to spawn node in the user space, hyphae required",
					out.Pair("node", nodeid),
					out.Pair("error", err.Error()))
			}
		} else {
			cmd := exec.Command(node.registry.Binary)
			err := cmd.Start()
			if err != nil {
				return error.New(
					error.Generic,
					error.Node,
					"failed to spawn node in the spore space",
					out.Pair("node", nodeid),
					out.Pair("error", err.Error()))
			}
		}
	} else {
		if node.manifest.Namespace == manifest.Spore {
			return error.New(
				error.Generic,
				error.Node,
				"failed to spawn node; node requires spore space",
				out.Pair("node", nodeid))
		} else {
			err := n.hyphae.Spawn(node.registry.Binary)
			if err != nil {
				return error.New(
					error.Generic,
					error.Node,
					"failed to spawn node in the user space",
					out.Pair("node", nodeid),
					out.Pair("error", err.Error()))
			}
		}
	}

	return nil
}

func (n *Nodes) Kill(nodeid string) *error.Error {
	nodeid = n.resolveId(nodeid)

	return error.New(
		error.Generic,
		error.Node,
		"kill not yet implemented",
		out.Pair("node", nodeid))
}

func (n *Nodes) GetState(nodeid string) ([]out.IOut, *error.Error) {
	nodeid = n.resolveId(nodeid)
	
	node, ok := n.nodes[nodeid]
	if !ok {
		return nil, error.New(
			error.Missing,
			error.Node,
			"cannot retrieve state; node not installed")
	}

	return node.state()
}

//
//
// private
// nodes call
// helpers
//

func (n *Nodes) handle() string {
	n.index++
	return fmt.Sprintf("nodes-%d", n.index)
}

func (n *Nodes) resolveId(nodeid string) string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	_, ok := n.nodes[nodeid]; 
	if ok {
		return nodeid
	}

	suffix := "." + nodeid
	for fullid := range n.nodes {
		if strings.HasSuffix(fullid, suffix) {
			return fullid
		}
	}

	return nodeid
}

func (n *Nodes) acceptInstallationWarning(m *manifest.Manifest) bool {
	if m.Trust == manifest.StandardTrust || m.Trust == manifest.Untrusted {
		return true
	}
	
	handle := n.handle()
	title := "Installation Warning"
	var description strings.Builder
	description.WriteString(fmt.Sprintf(`Do you accept [%s] as a node with a [%s] level of trust?`, m.ID, string(m.Trust)))
	description.WriteString(fmt.Sprintf(`\n\nThis will grant it permission to all capabilities and is unadvised unless you are absolutely sure of the source.`))
	raw := fmt.Sprintf(`dialog.alert title="%s" description="%s" entries=[ Grant Deny ] ~%s`, title, description.String(), handle)

	msg, err := message.Request(raw, m.ID)
	if err != nil {
		n.bus.WitnessSpore(fmt.Sprintf("Request creation failure (%s)", err.Error()))
		return false
	}
	ch := n.pending.Await(handle)
	err = n.bus.Request(msg)
	if err != nil {
		n.pending.Delete(handle)
		n.bus.WitnessSpore(fmt.Sprintf("Request sending failure (%s)", err.Error()))
		return false
	}

	response, err := n.pending.WaitFor(handle, ch)
	if err != nil {
		n.bus.WitnessSpore(fmt.Sprintf("Response failure (%s)", err.Error()))
		return false
	}
	if response.Flag("error") {
		n.bus.WitnessSpore(fmt.Sprintf("Response failure (%s)", err.Error()))
		return false
	}
	entry, err := response.Arg("entry")
	if err != nil {
		n.bus.WitnessSpore(fmt.Sprintf("Response failure (%s)", err.Error()))
		return false
	}
	return entry == "Grant"
}
