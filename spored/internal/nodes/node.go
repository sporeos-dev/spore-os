package nodes

import (
	"bufio"
	"errors"
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

	writer.WriteString("OK\n")
	writer.Flush()

	n.mu.Lock()
	n.conn = conn
	n.reader = reader
	n.writer = writer
	n.mu.Unlock()

	n.bus.Witness(
		message.Witness(
			"handshake successful",
			out.Pair("node", n.registry.ID)))

	go n.listen()
	return nil
}

func (n *node) listen() {

	for {
		raw, err := n.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				n.bus.Witness(
					message.Witness(
						"node disconnecting",
						out.Pair("node", n.registry.ID)))
				break
			} else {
				n.Receive(
					error.New(
						error.InitializationFailure,
						error.Node,
						"read failure",
						out.Pair("node", n.registry.ID)))
				continue
			}
		}
		raw = strings.TrimSpace(raw)
		// i := index
		// index++
		
		// witnessing starts with witness
		if strings.HasPrefix(raw, "witness") {
			
			body := strings.TrimPrefix(raw, "witness ")
			n.bus.Witness(message.Node(body, n.registry.ID))
			continue

		// publishing starts with publish
		} else if strings.HasPrefix(raw, "publish") {

			broadcast, ok := message.Broadcast(raw, n.registry.ID)
			if !ok {
				n.bus.Witness(
					message.Witness(
						"malformed broadcast",
						out.Flag("spore_incoming"),
						out.Pair("node", n.registry.ID),
						out.Pair("raw", raw)))
				n.Receive(error.New(
					error.Malformed,
					error.Node,
					"failed to publish",
					out.Pair("node", n.registry.ID),
					out.Pair("raw", raw)))
				continue
			}
			n.bus.Witness(broadcast)

			topic := broadcast.Capability()
			ok = false
			for _, el := range n.manifest.Topics {
				if topic == el.Name {
					ok = true
					break
				}
			}
			if !ok {
				n.Receive(error.New(
					error.NotPermitted,
					error.Node,
					"topic not announced in the manifest",
					out.Pair("node", n.registry.ID),
					out.Pair("topic", topic)).
					WithMessage(broadcast))
				continue
			}

			err := n.bus.Broadcast(broadcast)
			if err != nil {
				n.Receive(err)
			}

		// response starts with handle
		} else if strings.HasPrefix(raw, "~") {

			response, ok := message.Response(raw, n.registry.ID)
			if !ok {
				n.bus.Witness(
					message.Witness(
						"malformed response",
						out.Flag("spore_incoming"),
						out.Pair("node", n.registry.ID),
						out.Pair("raw", raw)))
				n.Receive(error.New(
					error.Malformed,
					error.Node,
					"failed to respond",
					out.Pair("node", n.registry.ID),
					out.Pair("raw", raw)))
				continue
			}
			n.bus.Witness(response)

			err := n.bus.Response(response)
			if err != nil {
				n.Receive(err)
			}

		// fallback to request
		} else {

			request, ok := message.Request(raw, n.registry.ID)
			if !ok {
				n.bus.Witness(
					message.Witness(
						"malformed request",
						out.Flag("spore_incoming"),
						out.Pair("node", n.registry.ID),
						out.Pair("raw", raw)))
				n.Receive(error.New(
					error.Malformed,
					error.Node,
					"failed to request",
					out.Pair("node", n.registry.ID),
					out.Pair("raw", raw)))
			}
			n.bus.Witness(request)

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

			err := n.bus.Request(request)
			if err != nil {
				n.Receive(err)
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
				out.Pair("node", n.registry.ID)).
				WithMessage(message)
		} else {
			return error.New(
				error.Generic,
				error.Node,
				"node not connected",
				out.Pair("node", n.registry.ID)).
				WithMessage(message)
		}
	}

	n.bus.Witness(message)
	n.writer.WriteString(message.Wire() + "\n")
	n.writer.Flush()

	return nil
}

func (n *node) Witness(message message.Message) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.writer == nil {
		return
	}

	n.writer.WriteString(message.Witness() + "\n")
	n.writer.Flush()
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
