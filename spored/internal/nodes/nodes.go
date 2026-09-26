// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"spored/internal/iface"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/registry"
	"spored/internal/utilities/await"
	"spored/internal/utilities/file"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"spored/internal/witness"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"spored/internal/utilities/error"
)

const handshakeTimeout = 5 * time.Second

// selfNodeId identifies Nodes' own pending requests (e.g. install confirmation) on the bus.
const selfNodeId = "nodes-in-spore"

// nodesSelf represents Nodes itself as a bus participant, distinct from the
// individually registered node connections, so responses to requests Nodes
// makes on its own behalf (e.g. install confirmation) can be routed back.
type nodesSelf struct {
	nodes *Nodes
}

func (s *nodesSelf) Id() string                             { return selfNodeId }
func (s *nodesSelf) IsConnected() bool                      { return true }
func (s *nodesSelf) IsWitness() bool                        { return false }
func (s *nodesSelf) GetManifest() *manifest.Manifest        { return nil }
func (s *nodesSelf) Receive(msg iface.Message) *error.Error { return s.nodes.pending.Receive(msg) }
func (s *nodesSelf) Witness(msg iface.Message)              {}

type Nodes struct {
	index   atomic.Int64
	pending *await.Pending

	registry *registry.Registry

	mu    sync.RWMutex
	nodes map[string]*node

	bus         ibus
	hyphae      ihyphae
	permissions ipermissions
	spore       ispore
}

func New() *Nodes {
	nodes := &Nodes{
		pending:  await.New(error.Node),
		registry: registry.New(),
		nodes:    make(map[string]*node),
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

	n.bus.Register(&nodesSelf{nodes: n})

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
				witness.Send(
					error.New(
						error.Generic,
						error.Node,
						"skipped autostart, requires hyphae",
						out.Pair("node", el.registry.ID)))
				continue
			}

			err := n.Spawn(el.registry.ID)
			if err != nil {
				witness.Send(
					error.New(
						error.Generic,
						error.Node,
						"autostart failed to spawn",
						out.Pair("node", el.registry.ID)))
				continue
			}

			witness.Send(
				message.Witness(
					"node autostarted",
					out.Pair("node", el.registry.ID)))
		}
	}
}

func (n *Nodes) HandleConnection(conn net.Conn) {

	erro := conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetReadDeadline(time.Time{}) //nolint:errcheck
	if erro != nil {
		n.bus.Witness(
			error.New(
				error.Generic, 
				error.Node,
				"unable to read deadline",
				out.Pair("error", erro.Error())))
	}

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	nodeid, erro := reader.ReadString('\n')
	if erro != nil {
		err := error.New(
			error.HandshakeDenial,
			error.Node,
			"failed handshake at step 1: initial read",
			out.Pair("error", erro.Error()))

		witness.Send(err)
		writer.Flush()
		conn.Close()
		return
	}

	nodeid = strings.TrimSpace(nodeid)
	witness.Send(
		message.Witness(
			"node connecting",
			out.Pair("node", nodeid)))

	n.mu.RLock()
	defer n.mu.RUnlock()
	node, ok := n.nodes[nodeid]
	if !ok {
		err := error.New(
			error.HandshakeDenial,
			error.Node,
			"node not installed",
			out.Pair("node", nodeid))

		witness.Send(err)
		writer.WriteString(err.Wire() + "\n")
		writer.Flush()
		conn.Close()
		return
	}

	// Handle the connection with the node
	err := node.handleConnection(conn, reader, writer, n.hyphae)
	if err != nil {
		witness.Send(err)
		writer.WriteString(err.Wire() + "\n")
		writer.Flush()
		conn.Close()
		return
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
	if !ok {
		return nil
	}

	if node.manifest.ID == "" && n.hyphae != nil {
		content, err := n.hyphae.ManifestRead(node.manifest.Path)
		if err == nil {
			node.manifest.LoadContent(content)

			// developer trust implicit
			if node.manifest.Trust == manifest.Developer {
				found := false
				for _, el := range n.registry.Elements {
					if el.ID == nodeid {
						if el.Checksum == string(manifest.Developer) {
							node.manifest.Status.Set(status.Verified)
						} else {
							node.manifest.Status.Set(status.FailedChecksum)
						}
						found = true
					}
				}
				if !found {
					node.manifest.Status.Set(status.FailedChecksum)
				}

				// otherwise checksum
			} else {
				checksum, err := n.hyphae.HashFile(node.manifest.Path)
				if err == nil {
					if node.manifest.ExpectedChecksum == checksum {
						node.manifest.Status.Set(status.Verified)
					} else {
						node.manifest.Status.Set(status.FailedChecksum)
					}
				}
			}

			n.bus.Register(node)
		}
	}

	return node.manifest
}

func (n *Nodes) Install(path string) *error.Error {

	witness.Send(
		message.Witness(
			"installing node",
			out.Pair("path", path)))

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

		granted, err := n.acceptInstallationWarning(manifest)
		if err != nil {
			return err
		}
		if !granted {
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

		err = n.registry.Add(registryElement)
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

		granted, err := n.acceptInstallationWarning(manifest)
		if err != nil {
			return err
		}
		if !granted {
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

	// freshly installed, so trust already vetted via acceptInstallationWarning
	n.permissions.RegisterNode(node.manifest)

	if len(node.manifest.Permissions) > 0 {
		for _, el := range node.manifest.Permissions {
			granted, err := n.permissions.Request(node.registry.ID, el.Name, el.Reasons)
			if err != nil {
				witness.Send(err)
				continue
			}
			if granted {
				witness.Send(
					message.Witness(
						"permission granted",
						out.Pair("node", node.registry.ID),
						out.Pair("capability", el.Name)))
			} else {
				witness.Send(
					message.Witness(
						"permission denied",
						out.Pair("node", node.registry.ID),
						out.Pair("capability", el.Name)))
			}
		}
	}

	return nil
}

func (n *Nodes) Uninstall(nodeid string) *error.Error {

	nodeid = n.resolveId(nodeid)
	witness.Send(
		message.Witness(
			"uninstalling node",
			out.Pair("node", nodeid)))

	err := n.Kill(nodeid)
	if err != nil {
		witness.Send(err)
		// do not return
		// log failure and continue
	}

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

	node.close()
	delete(n.nodes, nodeid)
	return n.registry.Remove(node.registry.ID)
}

func (n *Nodes) Spawn(nodeid string) *error.Error {

	nodeid = n.resolveId(nodeid)
	witness.Send(
		message.Witness(
			"spawning node",
			out.Pair("node", nodeid)))

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
				return err
			}
		} else {
			cmd := exec.Command(node.registry.Binary)
			err := cmd.Start()
			if err != nil {
				return error.New(
					error.Generic,
					error.Node,
					"failed to spawn node",
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
				return err
			}
		}
	}

	return nil
}

func (n *Nodes) Kill(nodeid string) *error.Error {

	nodeid = n.resolveId(nodeid)
	witness.Send(
		message.Witness(
			"killing node",
			out.Pair("node", nodeid)))

	node, ok := n.nodes[nodeid]
	if !ok {
		return error.New(
			error.Missing,
			error.Node,
			"cannot kill; node not active",
			out.Pair("node", nodeid))
	}

	pid, ok := node.ProcessID()
	if !ok {
		return error.New(
			error.Generic,
			error.Node,
			"failed to retrieve process ID for node",
			out.Pair("node", nodeid))
	}

	err := n.hyphae.Kill(pid)
	if err != nil {
		return err
	}
	return nil
}

func (n *Nodes) GetState(nodeid string) (*State, *error.Error) {
	nodeid = n.resolveId(nodeid)

	node, ok := n.nodes[nodeid]
	if !ok {
		return nil, error.New(
			error.Missing,
			error.Node,
			"cannot retrieve state; node not installed")
	}

	return node.state(), nil
}

//
//
// private
// nodes call
// helpers
//

func (n *Nodes) handle() string {
	return fmt.Sprintf("nodes-%d", n.index.Add(1))
}

func (n *Nodes) resolveId(nodeid string) string {
	n.mu.RLock()
	defer n.mu.RUnlock()

	_, ok := n.nodes[nodeid]
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

func (n *Nodes) acceptInstallationWarning(m *manifest.Manifest) (bool, *error.Error) {

	// e.g.
	// dev.sporeos.shell.confirm body="Do you accept [nodeid] at a trust level of [trust]?" lines=[ ... ]
	var conf strings.Builder
	conf.WriteString("dev.sporeos.shell.confirm body=\"Do you accept node [")
	conf.WriteString(m.ID)
	conf.WriteString("] at a trust level of [")
	conf.WriteString(string(m.Trust))
	conf.WriteString("]?\" lines=[ \"Important Considerations\", ")
	switch m.Trust {
	case manifest.System:
		conf.WriteString("\" - system trust implies full control and should only be granted to core system components.\", ")
		conf.WriteString("\" - caution and thorough vetting are required before granting system trust.\", ")
		conf.WriteString("\" - while you are free to proceed, we suggest denying this request\"")
	case manifest.Developer:
		conf.WriteString("\" - developer trust grants free permission to all capabilities in the mesh\", ")
		conf.WriteString("\" - caution is advised when granting developer trust.\", ")
		conf.WriteString("\" - only grant developer trust if you are the developer of the node you are installing.\"")
	case manifest.Trusted:
		conf.WriteString("\" - trusted trust implies a higher level of scrutiny and should be granted to nodes with a proven track record.\", ")
		conf.WriteString("\" - only grant trusted trust if you are confident in the node's reliability.\"")
		conf.WriteString("\" - trusted nodes require capability permissions to be granted individually.\"")
	case manifest.StandardTrust:
		conf.WriteString("\" - standard trust implies a baseline level of scrutiny and should be granted to nodes with a reasonable track record.\", ")
		conf.WriteString("\" - only grant standard trust if you are confident in the node's reliability.\"")
		conf.WriteString("\" - standard nodes require capability permissions to be granted individually.\"")
	case manifest.Untrusted:
		conf.WriteString("\" - untrusted nodes should be treated with caution and granted minimal permissions.\", ")
		conf.WriteString("\" - only grant untrusted trust if you are confident in the node's reliability.\"")
		conf.WriteString("\" - untrusted nodes require capability permissions to be granted individually.\"")
	}
	conf.WriteString(" ] ~")
	handle := n.handle()
	conf.WriteString(handle)
	raw := conf.String()

	msg, ok := message.Request(raw, selfNodeId)
	if !ok {
		err := error.New(
			error.Malformed,
			error.Node,
			"failed to ensure trust",
			out.Pair("node", m.ID))
		witness.Send(err)
		return false, err
	}

	ch := n.pending.Await(handle)
	err := n.bus.Request(msg)
	if err != nil {
		n.pending.Delete(handle)
		witness.Send(err)
		return false, err
	}

	response, err := n.pending.WaitFor(handle, ch)
	if err != nil {
		witness.Send(err)
		return false, err
	}
	if response.Flag("error") {
		return false, nil
	}
	return response.Flag("yes"), nil
}
