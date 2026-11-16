// Copyright 2026 Matt Harrison
// SPDX-License-Identifier: AGPL-3.0-only

package nodes

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"slices"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	node_event "spored/internal/event/node"
	registry_event "spored/internal/event/registry"
	sync_event "spored/internal/event/sync"
	"spored/internal/hyphae"
	"spored/internal/manifest"
	"spored/internal/node"
	"spored/internal/pal"
	reg_el "spored/internal/registry/element"
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
var _ registry_event.IRegistryAnnouncementHandler = (*Pool)(nil)
var _ registry_event.IRegistryWithdrawalHandler = (*Pool)(nil)
var _ message_event.IRequestHandler = (*Pool)(nil)

type Pool struct {
	*base.Base
	index   atomic.Int64
	pending *await.Pending

	mu    sync.RWMutex
	nodes map[string]*node.Single
}

type installation struct {
	manifest *manifest.Manifest
	registry *reg_el.Element
}

func New() *Pool {
	return &Pool{
		Base:     base.New().SetName("nodepool"),
		pending:  await.New(error.NodePool),
		nodes:    make(map[string]*node.Single),
	}
}

func (p *Pool) Open() {
	event.Subscribe(p)
}

func (p *Pool) Close() {
	event.Unsubscribe(p)
}

func (p *Pool) HandleConnection(conn net.Conn) {

	erro := conn.SetReadDeadline(time.Now().Add(handshakeTimeout))
	defer conn.SetReadDeadline(time.Time{}) //nolint:errcheck
	if erro != nil {
		witness.Error(error.HandshakeDenial, error.Node, "read deadline", out.Pair("error", erro.Error()))
	}

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	nodeid, erro := reader.ReadString('\n')
	if erro != nil {
		witness.Error(error.HandshakeDenial, error.Node, "handshake read failed", out.Pair("error", erro.Error()))
		writer.WriteString("FAIL handshake read failed\n")
		writer.Flush()
		conn.Close()
		return
	}

	nodeid = strings.TrimSpace(nodeid)
	witness.Spore("node connecting", out.Pair("node", nodeid))

	p.mu.RLock()
	defer p.mu.RUnlock()
	node, ok := p.nodes[nodeid]
	if !ok {
		witness.Error(error.HandshakeDenial, error.Node, "node not installed", out.Pair("node", nodeid))
		writer.WriteString("FAIL node not installed\n")
		writer.Flush()
		conn.Close()
		return
	}

	// Handle the connection with the node
	// failures will be handled in the function
	// (including witness & replies)
	node.HandleConnection(conn, reader, writer)
}

// registry announcment
func (p *Pool) OnRegistryAnnouncement(ctx registry_event.IRegistryContext) {
	el := ctx.Registry()
	witness.Spore("registry announcement", out.Pair("node", el.ID))

	{
		p.mu.RLock()
		_, ok := p.nodes[el.ID]
		p.mu.RUnlock()

		if ok {
			witness.Error(error.Collision, error.NodePool, "cannot add to nodepool", out.Pair("node", el.ID))
			return
		}

		node := node.New(el)

		p.mu.Lock()
		p.nodes[el.ID] = node		
		p.mu.Unlock()

		node.Autostart()
	}

	witness.Spore("node added to registry", out.Pair("node", el.ID))
}

// registry withdawal
func (p *Pool) OnRegistryWithdrawal(ctx registry_event.IRegistryContext) {
	el := ctx.Registry()
	witness.Spore("registry withdrawal", out.Pair("node", el.ID))

	{
		p.mu.Lock()

		node, ok := p.nodes[el.ID]
		if ok {
			node.Close()
			delete(p.nodes, el.ID)
			p.mu.Unlock()
			witness.Spore("node removed from registry", out.Pair("node", el.ID))
			return
		}

		p.mu.Unlock()
	}

	witness.Error(error.Missing, error.NodePool, "cannot withdraw from registry", out.Pair("node", el.ID))
}

//
//
// irequest
//

func (p *Pool) OnRequest(ctx message_event.IRequestContext) {

	command := ctx.Command()
	message := ctx.Message()
	origin := ctx.Origin()
	handle := ctx.Handle()
	
	go func() {
		respondError := func(code error.Code, what string) {
			response := message_event.ResponseError(code, error.NodePool, what, handle, command, origin)
			event.Dispatch().Message.Notify(message_event.Raw(response.Message(), "dev.sporeos.SPORE"))
		}
		
		pm, ok := cparser.ParseAndValidate(message)
		if !ok {
			response := message_event.ResponseError(error.Malformed, error.NodePool, "bad message", pm.Handle, pm.Command, origin)
			event.Dispatch().Response.Notify(response)
			return
		}

		if command == "install" || strings.HasSuffix(command, ".install") {
			
			path, ok := pm.Args["path"]
			if !ok {
				witness.Error(error.Missing, error.NodePool, "missing argument", out.Pair("arg", "path"))
				response := message_event.ResponseError(error.Missing, error.NodePool, "missing argument 'path'", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}

			ok = p.nodeInstall(path)
			if !ok {
				response := message_event.ResponseError(error.Generic, error.NodePool, "installation failure", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}

		} else if command == "uninstall" || strings.HasSuffix(command, ".uninstall") {

			node, ok := pm.Args["node"]
			if !ok {
				witness.Error(error.Missing, error.NodePool, "missing argument", out.Pair("arg", "node"))
				response := message_event.ResponseError(error.Missing, error.NodePool, "missing argument 'node'", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}

			ok = p.nodeUninstall(node)
			if !ok {
				response := message_event.ResponseError(error.Generic, error.NodePool, "uninstall failure", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}
			
		} else if command == "reinstall" || strings.HasSuffix(command, ".reinstall") {

			node, ok := pm.Args["node"]
			if !ok {
				witness.Error(error.Missing, error.NodePool, "missing argument", out.Pair("arg", "node"))
				response := message_event.ResponseError(error.Missing, error.NodePool, "missing argument 'node'", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}
			
			ok = p.nodeReinstall(node)
			if !ok {
				witness.Error(error.Generic, error.NodePool, "reinstall failed")
				response := message_event.ResponseError(error.Generic, error.NodePool, "reinstall failed", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}
			
		} else if command == "spawn" || strings.HasSuffix(command, ".spawn") {

			node, ok := pm.Args["node"]
			if !ok {
				witness.Error(error.Missing, error.NodePool, "missing argument", out.Pair("arg", "node"))
				respondError(error.Missing, "missing argument 'node'")
				return
			}

			ok = p.nodeSpawn(node)
			if !ok {
				witness.Error(error.Generic, error.NodePool, "spawn failed")
				respondError(error.Generic, "spawn failed")
				return
			}

		} else if command == "kill" || strings.HasSuffix(command, ".kill") {
			
			node, ok := pm.Args["node"]
			if !ok {
				witness.Error(error.Missing, error.NodePool, "missing argument", out.Pair("arg", "node"))
				respondError(error.Missing, "missing argument 'node'")
				return
			}

			ok = p.nodeKill(node)
			if !ok {
				witness.Error(error.Generic, error.NodePool, "kill failed")
				respondError(error.Generic, "kill failed")
				return
			}

		} else if command == "respawn" || strings.HasSuffix(command, ".respawn") {

			node, ok := pm.Args["node"]
			if !ok {
				witness.Error(error.Missing, error.NodePool, "missing argument", out.Pair("arg", "node"))
				response := message_event.ResponseError(error.Missing, error.NodePool, "missing argument 'node'", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}
			
			ok = p.nodeRespawn(node)
			if !ok {
				witness.Error(error.Generic, error.NodePool, "respawn failed")
				response := message_event.ResponseError(error.Generic, error.NodePool, "respawn failed", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}

		} else if command == "path" || strings.HasSuffix(command, ".path") {
			node, ok := pm.Args["node"]
			if !ok {
				witness.Error(error.Missing, error.NodePool, "missing argument", out.Pair("arg", "node"))
				response := message_event.ResponseError(error.Missing, error.NodePool, "missing argument 'node'", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}
			path, ok := p.nodePath(node)
			if !ok {
				witness.Error(error.RuntimeFailure, error.NodePool, "path failure")
				response := message_event.ResponseError(error.RuntimeFailure, error.NodePool, "path failure", pm.Handle, pm.Command, origin)
				event.Dispatch().Response.Notify(response)
				return
			}
			response := message_event.Raw(
				fmt.Sprintf("~%s:%s ok path='%s'", handle, command, path),
				"dev.sporeos.SPORE")
			event.Dispatch().Message.Notify(response)
			return

		} else {

			witness.Error(error.NotImplemented, error.NodePool, "unknown command")
			response := message_event.ResponseError(error.NotImplemented, error.NodePool, "unknown command", pm.Handle, pm.Command, origin)
			event.Dispatch().Response.Notify(response)
			return
		}

		response := message_event.Raw(
			fmt.Sprintf("~%s:%s ok", handle, command),
			"dev.sporeos.SPORE")
		event.Dispatch().Message.Notify(response)
	}()
}

func (p *Pool) Api() []string {
	return []string{
		"dev.sporeos.SPORE.node.install", 
		"dev.sporeos.SPORE.node.uninstall", 
		"dev.sporeos.SPORE.node.reinstall",
		"dev.sporeos.SPORE.node.spawn",
		"dev.sporeos.SPORE.node.kill",
		"dev.sporeos.SPORE.node.respawn",
		"dev.sporeos.SPORE.node.path",
	}
}

func (p *Pool) nodeInstall(path string) bool {
		var contents string
		var checksum string
		if file.IsReadable(path) {
			co, err := file.Read(path)
			if err != nil {
				witness.Error(error.Generic, error.NodePool, "unable to read manifest", out.Pair("path", path), out.Pair("err", err.Error()))
				return false
			}
			ch, err := file.CalculateChecksum(path)
			if err != nil {
				witness.Error(error.Generic, error.NodePool, "unable to checksum", out.Pair("path", path))
				return false
			}
			contents = co
			checksum = ch
		} else {
			co, ok := hyphae.ManifestRead(path)
			if !ok {
				witness.Error(error.Generic, error.NodePool, "unable to read manifest", out.Pair("path", path))
				return false
			}
			ch, ok := hyphae.FileHash(path)
			if !ok {
				return false
			}
			contents = co
			checksum = ch
		}
			
		m := &manifest.Manifest{
			Status: status.New(),
			Path: path,
			ExpectedChecksum: checksum,
		}
		err := m.LoadContent(contents)
		if err != nil {
			witness.Error(error.Malformed, error.NodePool, "invalid manifest", out.Pair("path", path), out.Pair("err", err.Error()))
			return false
		}

		// check if already exists
		p.mu.RLock()
		_, ok := p.nodes[m.ID]
		p.mu.RUnlock()
		if ok {
			witness.Error(error.Collision, error.NodePool, "node already exists", out.Pair("node", m.ID))
			return false
		}
		
		// handle trust
		switch m.Trust {
		case manifest.System:
			witness.Error(error.NotPermitted, error.NodePool, "cannot install system manifest")
			return false
		case manifest.Developer:
			raw := fmt.Sprintf("%s ~%s body='%s' lines=[ '%s', '%s', '%s', '%s' ]",
				"dev.sporeos.shell.confirm",
				makeHandle(),
				"Are you sure you want to install this manfiest with developer trust?",
				"Important",
				"- we suggest you decline this offer unless you absolutely understand the risks",
				"- only proceed if this node is your own node under development",
				"- third-party nodes should not be accepted with developer trust")
			
			sreq := sync_event.Request(raw, "dev.sporeos.SPORE")
			event.Dispatch().SynchronizedRequest.Notify(sreq)
			srep := sreq.Response()
			if srep == nil {
				witness.Error(error.Missing, error.NodePool, "no response to developer trust confirmation")
				return false
			}
			
			pm, ok := cparser.ParseAndValidate(srep.Message())
			if !ok {
				// witnessed in ParseAndValidate
				return false
			}
			if slices.Contains(pm.Flags, "error") || slices.Contains(pm.Flags, "cancelled") {
				// witnessed in Incoming response
				return false
			} else if slices.Contains(pm.Flags, "no") {
				witness.Spore("developer trust declined")
				return false
			} else if slices.Contains(pm.Flags, "yes") {
				witness.Spore("developer trust accepted")
				// accepted, continue on
			} else {
				witness.Error(error.Missing, error.NodePool, "response missing both 'yes' and 'no' flags")
				return false
			}
			
			m.ExpectedChecksum = "developer"

		case manifest.Trusted:
		case manifest.StandardTrust:
		case manifest.Untrusted:
		}

		man := node_event.New(m)
		event.Dispatch().NodeInstallation.Notify(man)

		if !man.Installed() {
			witness.Error(error.RuntimeFailure, error.NodePool, "registry installation failed")
			return false
		}
		return true
}

func (p *Pool) nodeUninstall(node string) bool {
	p.mu.RLock()
	n, ok := p.nodes[node]
	p.mu.RUnlock()
	if !ok {
		witness.Error(error.Missing, error.NodePool, "node not found", out.Pair("node", node))
		return false
	}

	p.nodeKill(node)
	n.Close()
	p.mu.Lock()
	delete(p.nodes, node)
	p.mu.Unlock()

	event.Dispatch().NodeUninstallation.Notify(node)
	return true
}

func (p *Pool) nodeReinstall(node string) bool {
	p.mu.RLock()
	single, ok := p.nodes[node]
	if !ok {
		witness.Error(error.Missing, error.NodePool, "node not found")
		p.mu.RUnlock()
		return false
	}
	m := single.Manifest()
	if m == nil {
		witness.Error(error.Missing, error.NodePool, "no manifest path")
		p.mu.RUnlock()
		return false
	}
	p.mu.RUnlock()

	ok = p.nodeUninstall(node)
	if !ok {
		witness.Error(error.Missing, error.NodePool, "uninstall failed")
		return false
	}
	return p.nodeInstall(m.Path)
}


func (p *Pool) nodeSpawn(nodeid string) bool {
	single, ok := p.findNode(nodeid)
	if !ok {
		return false
	}
	binary, namespace, ok := single.GetSpawnInfo()
	if !ok {
		witness.Error(error.Missing, error.NodePool, "missing node spawn information", out.Pair("node", nodeid))
		return false
	}

	switch namespace {
	case manifest.Hyphae:
		return hyphae.Hspawn(binary)
	case manifest.Spore:
		if !file.IsReadable(binary) {
			witness.Error(error.RuntimeFailure, error.NodePool, "spore-space binary is not readable", out.Pair("node", nodeid), out.Pair("path", binary))
			return false
		}
	default:
		witness.Error(error.Malformed, error.NodePool, "invalid node namespace", out.Pair("node", nodeid), out.Pair("namespace", string(namespace)))
		return false
	}

	cmd := exec.Command(binary)
	pal.ConfigureSpawnCommand(cmd)
	if err := cmd.Start(); err != nil {
		witness.Error(error.RuntimeFailure, error.NodePool, "failed to spawn node", out.Pair("node", nodeid), out.Pair("error", err.Error()))
		return false
	}
	go func() {
		if err := cmd.Wait(); err != nil {
			slog.Debug("spawned node exited", "node", nodeid, "error", err)
		}
	}()
	return true
}

func (p *Pool) nodeKill(nodeid string) bool {
	single, ok := p.findNode(nodeid)
	if !ok {
		return false
	}
	_, namespace, ok := single.GetSpawnInfo()
	if !ok {
		witness.Error(error.Missing, error.NodePool, "missing node information", out.Pair("node", nodeid))
		return false
	}
	pid, ok := single.ProcessID()
	if !ok {
		witness.Error(error.Missing, error.NodePool, "node has no connected process", out.Pair("node", nodeid))
		return false
	}

	switch namespace {
	case manifest.Spore:
		return killLocal(nodeid, pid)
	case manifest.Hyphae:
		return hyphae.HKill(pid)
	default:
		witness.Error(error.Malformed, error.NodePool, "invalid node namespace", out.Pair("node", nodeid), out.Pair("namespace", string(namespace)))
		return false
	}
}

func (p *Pool) nodeRespawn(node string) bool {
	ok := p.nodeKill(node)
	if !ok {
		witness.Error(error.RuntimeFailure, error.NodePool, "failed to kill node")
		return false
	}
	return p.nodeSpawn(node)
}

func (p *Pool) nodePath(node string) (string, bool) {
	single, ok := p.findNode(node)
	if !ok {
		witness.Error(error.Missing, error.NodePool, "node not found")
		return "", false
	}
	m := single.Manifest()
	if m == nil {
		witness.Error(error.Missing, error.NodePool, "no manifest")
		return "", false
	}
	return m.GetBinaryPath(), true
}

//
//
// private
// helper
//

func (p *Pool) findNode(nodeid string) (*node.Single, bool) {
	p.mu.RLock()
	if single, ok := p.nodes[nodeid]; ok {
		p.mu.RUnlock()
		return single, true
	}
	if nodeid == "" {
		p.mu.RUnlock()
		witness.Error(error.Missing, error.NodePool, "node not found", out.Pair("node", nodeid))
		return nil, false
	}

	var match *node.Single
	for id, single := range p.nodes {
		if strings.HasSuffix(id, "."+nodeid) {
			if match != nil {
				p.mu.RUnlock()
				witness.Error(error.Collision, error.NodePool, "ambiguous node", out.Pair("node", nodeid))
				return nil, false
			}
			match = single
		}
	}
	p.mu.RUnlock()
	if match == nil {
		witness.Error(error.Missing, error.NodePool, "node not found", out.Pair("node", nodeid))
		return nil, false
	}
	return match, true
}

func killLocal(nodeid string, pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		witness.Error(error.RuntimeFailure, error.NodePool, "failed to find node process", out.Pair("node", nodeid), out.Pair("error", err.Error()))
		return false
	}
	defer process.Release()
	if err := process.Kill(); err != nil {
		witness.Error(error.RuntimeFailure, error.NodePool, "failed to kill node process", out.Pair("node", nodeid), out.Pair("error", err.Error()))
		return false
	}
	return true
}

var index atomic.Int64
func makeHandle() string {
	return fmt.Sprintf("nodepool-%d", index.Add(1))
}
