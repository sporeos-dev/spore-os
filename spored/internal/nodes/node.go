package nodes

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"spored/internal/manifest"
	"spored/internal/message"
	"spored/internal/registry"
	"spored/internal/utilities/error"
	"spored/internal/utilities/out"
	"spored/internal/utilities/status"
	"strings"
	"sync"
)

type spawner interface {
	Spawn(nodeid string) *error.Error
}

type permitter interface {
	Can(nodeid string, capability string) bool
}

type node struct {
	registry *registry.Element
	manifest *manifest.Manifest
	bus ibus
	spawner spawner
	permitter permitter

	mu sync.RWMutex
	jsonMu sync.Mutex
	jsonHandles map[string]bool
	conn net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
}

//
//
// ctor/dtor
//

func newNode(registry *registry.Element, manifest *manifest.Manifest) *node {

	n := &node{
		registry: registry,
		manifest: manifest,
		bus: nil,
		jsonHandles: make(map[string]bool),
	}

	n.manifest.Verify()
	n.manifest.Load()

	return n
}

func (n *node) close() {
	n.bus.Unregister(n)
}

func (n *node) set(bus ibus, spawner spawner, permitter permitter) {
	n.spawner = spawner
	n.bus = bus
	n.permitter = permitter
	n.bus.Register(n)
}

func (n *node) state() ([]out.IOut, *error.Error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	hasConn := "true"
	if n.conn == nil {
		hasConn = "false"
	}

	outs := make([]out.IOut, 0)
	outs = append(outs, out.Pair("connected", hasConn))
	return outs, nil
}

//
//
// inode
//

func (n *node) handleConnection(conn net.Conn, reader *bufio.Reader, writer *bufio.Writer, hyphae ihyphae) *error.Error {

	err := n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.manifest.Verify()
	if n.manifest.Status.Get() == status.RequiresUserSpace {
		checksum, err := hyphae.HashFile(n.manifest.Path)
		if err != nil {
			return err
		}
		if checksum == n.manifest.ExpectedChecksum {
			n.manifest.Status.Set(status.Verified)
		} else {
			n.manifest.Status.Set(status.FailedChecksum)
			return error.New(
				error.HandshakeDenial,
				error.Node,
				"failed manifest verification",
				out.Pair("checksum", checksum),
				out.Pair("expected", n.manifest.ExpectedChecksum))
		}
	}
	err = n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.manifest.Load()
	err = n.checkManifestForFailure()
	if err != nil {
		return err
	}

	n.bus.Register(n)

	if n.manifest.Status.Get() != status.Verified {
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"unknown manifest verification failure",
			out.Pair("node", n.registry.ID),
			out.Pair("manifest_status", n.manifest.Status.String()))
	}

	b := newBinary(n.registry)
	err = n.checkBinaryForFailure(b)
	if err != nil {
		return err
	}

	b.verify(conn)
	if b.status.Get() == status.RequiresUserSpace {
		checksum, err := hyphae.HashFile(n.registry.Binary)
		if err != nil {
			return err
		}
		if checksum == n.registry.BinaryChecksum {
			b.status.Set(status.Verified)
		} else {
			b.status.Set(status.FailedChecksum)
			return error.New(
				error.HandshakeDenial,
				error.Node,
				"failed manifest verification",
				out.Pair("checksum", checksum),
				out.Pair("expected", n.registry.BinaryChecksum))
		}
	}
	err = n.checkBinaryForFailure(b)
	if err != nil {
		return err
	}

	if b.status.Get() != status.Verified {
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"unknown binary verification failure",
			out.Pair("node", n.registry.ID),
			out.Pair("binary_status", b.status.String()))
	}

	n.bus.WitnessOut("OK", n.registry.ID)
	writer.WriteString("OK\n")
	writer.Flush()

	n.mu.Lock()
	n.conn = conn
	n.reader = reader
	n.writer = writer
	n.mu.Unlock()

	n.bus.WitnessSpore(fmt.Sprintf("%s connected after successful handshake (%s)", n.registry.Name, n.registry.ID))

	go n.listen()
	return nil
}

func (n *node) listen() {

	for {
		raw, err := n.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				n.bus.WitnessSpore(fmt.Sprintf("%s disconnecting (%s)", n.registry.Name, n.registry.ID))
				break
			} else {
				n.bus.WitnessSpore(fmt.Sprintf("%s failed to read (%s, err %s", n.registry.Name, n.registry.ID, err.Error()))
				continue
			}
		}
		raw = strings.TrimSpace(raw)
		// i := index
		// index++
		
		// witnessing starts with witness
		if strings.HasPrefix(raw, "witness") {
			
			body := strings.TrimPrefix(raw, "witness ")
			n.bus.WitnessNode(body, n.registry.ID)
			continue

		// publishing starts with publish
		} else if strings.HasPrefix(raw, "publish") {

			n.bus.WitnessIn(raw, n.registry.ID)
			broadcast, err := message.Broadcast(raw, n.registry.ID)
			if err != nil {
				n.Error(err, "", "")
				continue
			}
			
			topic := broadcast.Topic()
			ok := false
			for _, el := range n.manifest.Topics {
				if topic == el.Name {
					ok = true
					break
				}
			}
			if !ok {
				n.Error(error.New(
					error.NotPermitted,
					error.Node,
					"topic not announced in the manifest",
					out.Pair("node", n.registry.ID),
					out.Pair("topic", topic)), "", "")
				continue
			}

			if err := n.bus.Broadcast(broadcast); err != nil {
				n.Error(err, "", "")
			}

		// response starts with handle
		} else if strings.HasPrefix(raw, "~") {

			n.bus.WitnessIn(raw, n.registry.ID)
			response, err := message.Response(raw, n.registry.ID)
			if err != nil {
				n.Error(err, "", "")
				continue
			}
			if err := n.bus.Response(response); err != nil {
				n.Error(err, response.Handle(), response.Command())
			}

		// fallback to request
		} else {

			n.bus.WitnessIn(raw, n.registry.ID)
			msg, err := message.Request(raw, n.registry.ID)
			if err != nil {
				n.Error(err, "", "")
				continue
			}
			
			// if n.manifest.Trust == manifest.StandardTrust || n.manifest.Trust == manifest.Untrusted {
			// 	command := msg.Command()
			// 	if !n.permitter.Can(n.registry.ID, command) {
			// 		n.Error(error.New(
			// 			error.NotPermitted,
			// 			error.Node,
			// 			"permission required",
			// 			out.Pair("node", n.registry.ID),
			// 			out.Pair("capability", command)), "", "")
			// 		return
			// 	}
			// }

			if err := n.bus.Request(msg); err != nil {
				n.Error(err, msg.Handle(), msg.Command())
			} else if msg.Flag("json") {
				n.jsonMu.Lock()
				n.jsonHandles[msg.Handle()] = true
				n.jsonMu.Unlock()
			}
		}
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	n.reader = nil
	n.writer = nil
	n.conn.Close()
}

//
//
// inode
//

func (n *node) Id() string {
	return n.registry.ID
}

func (n *node) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.conn != nil
}

func (n *node) IsWitness() bool {
	if n.manifest == nil {
		return false
	}
	return n.manifest.Witness
}

func (n *node) GetManifest() *manifest.Manifest {
	return n.manifest
}

func (n *node) Receive(message message.Message) *error.Error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.writer == nil {
		if n.manifest.Launch == manifest.Lazy {
			return error.New(
				error.Generic,
				error.Node,
				"lazy spawning not yet implemented",
				out.Pair("node", n.registry.ID))
		} else {
			return error.New(
				error.Generic,
				error.Node,
				"node not connected",
				out.Pair("node", n.registry.ID))
		}
	}

	wire := message.Get()
	if !message.IsWitness() {
		n.jsonMu.Lock()
		wantsJSON := n.jsonHandles[message.Handle()]
		delete(n.jsonHandles, message.Handle())
		n.jsonMu.Unlock()
		if wantsJSON {
			wire = message.ToJSON()
		}
	}

	n.writer.WriteString(wire + "\n")
	n.writer.Flush()

	if !message.IsWitness() {
		n.bus.WitnessOut(message.Get(), n.registry.ID)
	}

	return nil
}

func (n *node) Error(err *error.Error, handle string, subject string) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	var wire string
	if handle != "" {
		wire = fmt.Sprintf("~%s:%s %s", handle, subject, err.Wire())
	} else {
		wire = err.Wire()
	}

	n.bus.WitnessOut(fmt.Sprintf("%s failed to route (%s, err %s)", n.registry.Name, n.registry.ID, err.Witness()), n.registry.ID)
	n.writer.WriteString(wire + "\n")
	n.writer.Flush()

	n.bus.WitnessOut(wire, n.registry.ID)
}

//
//
// private
//

func (n *node) checkManifestForFailure() *error.Error {
	s := n.manifest.Status.Get()
	switch s {
	case status.Missing:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"missing manifest",
			out.Pair("node", n.registry.ID))
	case status.FailedChecksum:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"invalid manifest",
			out.Pair("node", n.registry.ID),
			out.Pair("expected_checksum", n.manifest.ExpectedChecksum))
	}
	return nil
}

func (n *node) checkBinaryForFailure(b *binary) *error.Error {
	s := b.status.Get()
	switch s {
	case status.Missing:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"missing binary",
			out.Pair("node", n.registry.ID),
			out.Pair("path", b.path))
	case status.FailedChecksum:
		return error.New(
			error.HandshakeDenial,
			error.Spore,
			"failed checksum",
			out.Pair("node", n.registry.ID),
			out.Pair("expected_checksum", b.expectedChecksum))
	}
	return nil
}
